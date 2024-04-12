package main

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/charmbracelet/log"
)

var oses = []string{"linux", "darwin", "windows"}
var arches = []string{"amd64", "arm64"}
var owner = "jumppad-labs"
var repo = "jumppad"

func New() *JumppadCI {
	return &JumppadCI{}
}

type JumppadCI struct {
	lastError          error
	goModCacheVolume   *CacheVolume
	goBuildCacheVolume *CacheVolume
}

func (d *JumppadCI) WithGoModCache(cache *CacheVolume) *JumppadCI {
	d.goModCacheVolume = cache
	return d
}

func (d *JumppadCI) WithGoBuildCache(cache *CacheVolume) *JumppadCI {
	d.goBuildCacheVolume = cache
	return d
}

func (d *JumppadCI) All(
	ctx context.Context,
	src *Directory,
	// +optional
	quick bool,
	// +optional
	githubToken *Secret,
	// +optional
	notorizeCert *File,
	// +optional
	notorizeCertPassword *Secret,
	// +optional
	notorizeKey *File,
	// +optional
	notorizeId string,
	// +optional
	notorizeIssuer string,
) (*Directory, error) {
	// if quick, only build for the current architecture
	if quick {
		d.setArchLocalMachine(ctx)
	}

	// get the version
	version := "0.0.0"
	sha := ""

	var output *Directory

	// remove the build output directory from the source
	src = src.
		WithoutDirectory(".dagger").
		WithoutDirectory("build-output")

	// if we have a github token, get the version from the associated PR label
	if githubToken != nil {
		version, sha, _ = d.getVersion(ctx, githubToken, src)
	}

	log.Info("Building version", "semver", version, "sha", sha)

	// run the unit tests
	d.UnitTest(ctx, src, !quick)

	// build the applications
	output, _ = d.Build(ctx, src, version, sha)

	// package the build outputs
	output, _ = d.Package(ctx, output, version)

	// create the archives
	output, _ = d.Archive(ctx, output, version)

	// if we have the notorization details sign and notorize the osx binaries
	if notorizeCert != nil && notorizeCertPassword != nil && notorizeKey != nil && notorizeId != "" && notorizeIssuer != "" {
		output, _ = d.SignAndNotorize(ctx, version, output, notorizeCert, notorizeCertPassword, notorizeKey, notorizeId, notorizeIssuer)
	}

	// generate the checksums
	output, _ = d.GenerateChecksums(ctx, output, version)

	return output, d.lastError
}

func (d *JumppadCI) Release(
	ctx context.Context,
	src *Directory,
	archives *Directory,
	githubToken *Secret,
	gemfuryToken *Secret,
) (string, error) {
	// create a new github release
	version, _ := d.GithubRelease(ctx, src, archives, githubToken)

	// update the brew formula at jumppad-labs/homebrew-repo
	d.UpdateBrew(ctx, version, githubToken)

	//	update the gemfury repository
	d.UpdateGemFury(ctx, version, gemfuryToken, archives)

	// update latest version on website
	d.UpdateWebsite(ctx, version, githubToken)

	return version, d.lastError
}

func (d *JumppadCI) UnitTest(
	ctx context.Context,
	src *Directory,
	withRace bool,
) error {
	if d.hasError() {
		return d.lastError
	}

	cli := dag.Pipeline("unit-test")

	raceFlag := ""
	if withRace {
		raceFlag = "-race"
	}

	golang := cli.Container().
		From("golang:latest").
		WithDirectory("/src", src).
		WithMountedCache("/go/pkg/mod", d.goModCache()).
		WithMountedCache("/root/.cache/go-build", d.goBuildCache()).
		WithWorkdir("/src").
		WithExec([]string{"go", "test", "-v", raceFlag, "./..."})

	_, err := golang.Sync(ctx)
	if err != nil {
		d.lastError = err
	}

	return err
}

func (d *JumppadCI) Build(
	ctx context.Context,
	src *Directory,
	version,
	sha string,
) (*Directory, error) {
	if d.hasError() {
		return nil, d.lastError
	}

	cli := dag.Pipeline("build")

	// create empty directory to put build outputs
	outputs := cli.Directory()

	// get `golang` image
	golang := cli.Container().
		From("golang:latest").
		WithDirectory("/src", src).
		WithWorkdir("/src").
		WithMountedCache("/root/.cache/go-build", d.goBuildCache()).
		WithMountedCache("/go/pkg/mod", d.goModCache())

	for _, goos := range oses {
		for _, goarch := range arches {
			fmt.Println("Build for", goos, goarch, "...")

			// create a directory for each os and arch
			path := fmt.Sprintf("build/%s/%s/", goos, goarch)

			// set GOARCH and GOOS in the build environment
			build, err := golang.
				WithEnvVariable("CGO_ENABLED", "0").
				WithEnvVariable("GOOS", goos).
				WithEnvVariable("GOARCH", goarch).
				WithExec([]string{
					"go", "build",
					"-o", path,
					"-ldflags", fmt.Sprintf("-X main.version=%s -X main.sha=%s", version, sha),
				}).
				Sync(ctx)

			if err != nil {
				d.lastError = err
				return nil, err
			}

			// get reference to build output directory in container
			outputs = outputs.WithDirectory(path, build.Directory(path))
		}
	}

	return outputs, nil
}

func (d *JumppadCI) Package(
	ctx context.Context,
	binaries *Directory,
	version string,
) (*Directory, error) {
	if d.hasError() {
		return nil, d.lastError
	}

	cli := dag.Pipeline("package")

	for _, os := range oses {
		if os == "linux" {
			for _, a := range arches {
				// create a package directory including the binaries
				pkg := cli.Directory()
				pkg = pkg.WithFile("/bin/jumppad", binaries.File(fmt.Sprintf("/build/linux/%s/jumppad", a)))

				// create a debian package
				p := cli.Deb().Build(pkg, a, "jumppad", version, "Nic Jackson", "Jumppad application")

				// add the debian package to the binaries directory
				binaries = binaries.WithFile(fmt.Sprintf("/pkg/linux/%s/jumppad.deb", a), p)
			}
		}
	}

	return binaries, nil
}

type Archive struct {
	// path of the
	Path   string
	Type   string
	Output string
}

var archives = []Archive{
	{Path: "/build/windows/amd64/jumppad.exe", Type: "zip", Output: "jumppad_%%VERSION%%_windows_x86_64.zip"},
	{Path: "/build/darwin/amd64/jumppad", Type: "zip", Output: "jumppad_%%VERSION%%_darwin_x86_64.zip"},
	{Path: "/build/darwin/arm64/jumppad", Type: "zip", Output: "jumppad_%%VERSION%%_darwin_arm64.zip"},
	{Path: "/build/linux/amd64/jumppad", Type: "targz", Output: "jumppad_%%VERSION%%_linux_x86_64.tar.gz"},
	{Path: "/build/linux/arm64/jumppad", Type: "targz", Output: "jumppad_%%VERSION%%_linux_arm64.tar.gz"},
	{Path: "/pkg/linux/amd64/jumppad.deb", Type: "copy", Output: "jumppad_%%VERSION%%_linux_x86_64.deb"},
	{Path: "/pkg/linux/arm64/jumppad.deb", Type: "copy", Output: "jumppad_%%VERSION%%_linux_arm64.deb"},
}

// Archive creates zipped and tar archives of the binaries
func (d *JumppadCI) Archive(
	ctx context.Context,
	binaries *Directory,
	version string,
) (*Directory, error) {
	if d.hasError() {
		return nil, d.lastError
	}

	cli := dag.Pipeline("archive")
	out := cli.Directory()

	zipContainer := cli.Container().
		From("alpine:latest").
		WithExec([]string{"apk", "add", "zip"})

	checksums := strings.Builder{}

	for _, a := range archives {
		outPath := strings.ReplaceAll(a.Output, "%%VERSION%%", version)
		switch a.Type {
		case "zip":
			// create a zip archive

			// first get the filename as with windows this has an extension
			fn := path.Base(a.Path)

			// zip the file
			zip := zipContainer.
				WithMountedFile(fn, binaries.File(a.Path)).
				WithExec([]string{"zip", "-r", outPath, fn})

			out = out.WithFile(outPath, zip.File(outPath))
		case "targz":
			// create a zip archive
			zip := zipContainer.
				WithMountedFile("/jumppad", binaries.File(a.Path)).
				WithExec([]string{"tar", "-czf", outPath, "/jumppad"})

			out = out.WithFile(outPath, zip.File(outPath))
		case "copy":
			out = out.WithFile(outPath, binaries.File(a.Path))
		}

		// generate the checksum
		cs, err := cli.Checksum().CalculateFromFile(ctx, out.File(outPath))
		if err != nil {
			d.lastError = fmt.Errorf("unable to generate checksum for archive: %w", err)
			return nil, d.lastError
		}

		// checksum is returned as "checksum filename" we need to remove the filename as it is not
		// the same as the release name
		csParts := strings.Split(cs, " ")

		checksums.WriteString(fmt.Sprintf("%s  %s\n", csParts[0], outPath))
	}

	out = out.WithNewFile("checksums.txt", checksums.String())

	return out, nil
}

func (d JumppadCI) GenerateChecksums(
	ctx context.Context,
	files *Directory,
	version string,
) (*Directory, error) {
	cli := dag.Pipeline("generate-checksums")
	checksums := strings.Builder{}

	for _, a := range archives {
		outPath := strings.ReplaceAll(a.Output, "%%VERSION%%", version)

		// generate the checksum
		cs, err := cli.Checksum().CalculateFromFile(ctx, files.File(outPath))
		if err != nil {
			d.lastError = fmt.Errorf("unable to generate checksum for archive: %w", err)
			return nil, d.lastError
		}

		// checksum is returned as "checksum filename" we need to remove the filename as it is not
		// the same as the release name
		csParts := strings.Split(cs, " ")

		checksums.WriteString(fmt.Sprintf("%s  %s\n", csParts[0], outPath))
	}

	files = files.WithNewFile("checksums.txt", checksums.String())
	return files, nil
}

var notorize = []Archive{
	{Path: "/jumppad_%%VERSION%%_darwin_x86_64.zip", Type: "zip", Output: "/jumppad_%%VERSION%%_darwin_x86_64.zip"},
	{Path: "/jumppad_%%VERSION%%_darwin_arm64.zip", Type: "zip", Output: "/jumppad_%%VERSION%%_darwin_arm64.zip"},
}

// SignAndNotorize signs and notorizes the osx binaries using the Apple notary service
func (d JumppadCI) SignAndNotorize(
	ctx context.Context,
	version string,
	archives *Directory,
	cert *File,
	password *Secret,
	key *File,
	keyId,
	keyIssuer string,
) (*Directory, error) {
	if d.hasError() {
		return nil, d.lastError
	}

	cli := dag.Pipeline("notorize")

	not := dag.Notorize().
		WithP12Cert(cert, password).
		WithNotoryKey(key, keyId, keyIssuer)

	out := archives

	zipContainer := cli.Container().
		From("alpine:latest").
		WithExec([]string{"apk", "add", "zip"})

	for _, a := range notorize {
		path := strings.ReplaceAll(a.Output, "%%VERSION%%", version)

		jpFile := zipContainer.
			WithMountedFile("/jumppad.zip", archives.File(path)).
			WithExec([]string{"unzip", "/jumppad.zip"}).
			File("/jumppad")

		notorized := not.SignAndNotorize(jpFile)

		nFile := zipContainer.
			WithMountedFile("/jumppad", notorized).
			WithExec([]string{"zip", "-r", path, "/jumppad"}).
			File(path)

		out = out.WithFile(path, nFile)
	}

	return out, nil
}

func (d *JumppadCI) GithubRelease(
	ctx context.Context,
	src *Directory,
	archives *Directory,
	githubToken *Secret,
) (string, error) {
	if d.hasError() {
		return "", d.lastError
	}

	version, sha, err := d.getVersion(ctx, githubToken, src)
	if err != nil {
		d.lastError = err
		return "", err
	}

	if version == "0.0.0" {
		d.lastError = fmt.Errorf("no version to release, did you tag the PR?")
		return "", d.lastError
	}

	cli := dag.Pipeline("release")

	_, err = cli.Github().
		WithToken(githubToken).
		CreateRelease(ctx, owner, repo, version, sha, GithubCreateReleaseOpts{Files: archives})

	if err != nil {
		d.lastError = err
		return "", err
	}

	return version, err
}

func (d *JumppadCI) UpdateBrew(
	ctx context.Context,
	version string,
	githubToken *Secret,
) error {
	if d.hasError() {
		return d.lastError
	}

	cli := dag.Pipeline("update-brew")

	_, err := cli.Brew().Formula(
		ctx,
		"https://jumppad.dev",
		"jumppad-labs/homebrew-repo",
		version,
		"Mr Jumppad",
		"hello@jumppad.dev",
		"jumppad",
		githubToken,
		BrewFormulaOpts{
			DarwinX86Url:   fmt.Sprintf("https://github.com/jumppad-labs/jumppad/releases/download/%s/jumppad_%s_darwin_x86_64.zip", version, version),
			DarwinArm64Url: fmt.Sprintf("https://github.com/jumppad-labs/jumppad/releases/download/%s/jumppad_%s_darwin_arm64.zip", version, version),
			LinuxX86Url:    fmt.Sprintf("https://github.com/jumppad-labs/jumppad/releases/download/%s/jumppad_%s_linux_x86_64.tar.g", version, version),
			LinuxArm64Url:  fmt.Sprintf("https://github.com/jumppad-labs/jumppad/releases/download/%s/jumppad_%s_linux_arm64.tar.giz", version, version),
		},
	)

	if err != nil {
		d.lastError = err
	}

	return err
}

var gemFury = []Archive{
	{Path: "/pkg/linux/amd64/jumppad.deb", Type: "copy", Output: "jumppad_%%VERSION%%_linux_x86_64.deb"},
	{Path: "/pkg/linux/arm64/jumppad.deb", Type: "copy", Output: "jumppad_%%VERSION%%_linux_arm64.deb"},
}

func (d *JumppadCI) UpdateGemFury(
	ctx context.Context,
	version string,
	gemFuryToken *Secret,
	archives *Directory,
) error {
	cli := dag.Pipeline("update-gem-fury")

	tkn, _ := gemFuryToken.Plaintext(ctx)
	url := fmt.Sprintf("https://%s@push.fury.io/jumppad/", tkn)

	for _, a := range gemFury {
		output := strings.Replace(a.Output, "%%VERSION%%", version, 1)

		_, err := cli.Container().
			From("curlimages/curl:latest").
			WithFile(output, archives.File(output)).
			WithExec([]string{"-F", fmt.Sprintf("package=@%s", output), url}).
			Sync(ctx)

		if err != nil {
			d.lastError = err
			return err
		}
	}

	return nil
}

func (d *JumppadCI) UpdateWebsite(
	ctx context.Context,
	version string,
	githubToken *Secret,
) error {
	cli := dag.Pipeline("update-website")

	f := cli.Directory().WithNewFile("version", version).File("version")

	_, err := cli.Github().
		WithToken(githubToken).
		CommitFile(
			ctx,
			"jumppad-labs", "jumppad-labs.github.io",
			"Mr Jumppad", "hello@jumppad.dev",
			"./public/latest",
			fmt.Sprintf("Update latest version: %s", version),
			f,
		)

	if err != nil {
		d.lastError = fmt.Errorf("failed to update website: %w", err)
		return d.lastError
	}

	return nil
}

var functionalTests = []string{
	"/build",
	"/certificates",
	"/container",
	"/docs",
	"/exec",
	"/multiple_k3s_clusters",
	"/nomad",
	"/single_file",
	"/single_k3s_cluster",
	"/terraform",
}

var runtimes = []string{"docker", "podman"}

type job struct {
	workingDirectory string
	runtime          string
}

func (d *JumppadCI) FunctionalTestAll(
	ctx context.Context,
	jumppad *File,
	src *Directory,
) error {
	if d.hasError() {
		return d.lastError
	}

	cli := dag.Pipeline("functional-test-all")

	// get the architecture of the current machine
	platform, err := cli.DefaultPlatform(ctx)
	if err != nil {
		panic(err)
	}

	jobCount := len(functionalTests) * len(runtimes)
	arch := strings.Split(string(platform), "/")[1]
	jobs := make(chan job, jobCount)
	errors := make(chan error, jobCount)

	// start the workers
	for w := 0; w < 3; w++ {
		go startTestWorker(ctx, w+1, cli, jumppad, src, arch, jobs, errors)
	}

	// add the jobs
	for _, runtime := range runtimes {
		for _, ft := range functionalTests {
			jobs <- job{workingDirectory: ft, runtime: runtime}
		}
	}
	close(jobs)

	for i := 0; i < jobCount; i++ {
		err := <-errors
		if err != nil {
			d.lastError = err
		}
	}

	return d.lastError
}

// FunctionalTest runs the functional tests for the jumppad binary
//
// example usage: dagger call functional-test --jumppad /path/to/jumppad --src /path/to/tests --working-directory /simple --runtime docker
func (d *JumppadCI) FunctionalTest(
	ctx context.Context,
	// path to the jumppad binary
	jumppad *File,
	// source directory containing the tests
	src *Directory,
	// working directory for the tests, relative to the source directory
	WorkingDirectory,
	// runtime to use for the tests, either docker or podman
	Runtime string,
) error {
	if d.hasError() {
		return d.lastError
	}

	wd := strings.TrimPrefix(WorkingDirectory, "/")
	pl := dag.Pipeline("functional-test-" + wd + "-" + Runtime)

	// get the architecture of the current machine
	platform, err := pl.DefaultPlatform(ctx)
	if err != nil {
		panic(err)
	}

	arch := strings.Split(string(platform), "/")[1]

	_, err = pl.Jumppad().
		TestBlueprintWithBinary(
			ctx,
			src,
			jumppad,
			JumppadTestBlueprintWithBinaryOpts{WorkingDirectory: WorkingDirectory, Architecture: arch, Runtime: Runtime, Cache: Runtime},
		)

	if err != nil {
		d.lastError = err
		return err
	}

	return nil
}

func (d *JumppadCI) getVersion(ctx context.Context, token *Secret, src *Directory) (string, string, error) {
	if d.hasError() {
		return "", "", d.lastError
	}

	cli := dag.Pipeline("get-version")

	// get the latest git sha from the source
	ref, err := cli.Container().
		From("alpine/git").
		WithDirectory("/src", src).
		WithWorkdir("/src").
		WithExec([]string{"rev-parse", "HEAD"}).
		Stdout(ctx)

	if err != nil {
		d.lastError = err
		return "", "", err
	}

	// make sure there is no whitespace from the output
	ref = strings.TrimSpace(ref)
	log.Info("github reference", "sha", ref)

	// get the next version from the associated PR label
	v, err := cli.Github().
		WithToken(token).
		NextVersionFromAssociatedPrlabel(ctx, owner, repo, ref)

	if err != nil {
		d.lastError = err
		return "", "", err
	}

	// if there is no version, default to 0.0.0
	if v == "" {
		v = "0.0.0"
	}

	log.Info("new version", "semver", v)

	return v, ref, nil
}

// gets the go build cache volume, if it doesn't exist it creates it
func (d *JumppadCI) goBuildCache() *CacheVolume {
	if d.goBuildCacheVolume == nil {
		d.goBuildCacheVolume = dag.CacheVolume("go-build-cache")
	}

	return d.goBuildCacheVolume
}

func (d *JumppadCI) goModCache() *CacheVolume {
	if d.goModCacheVolume == nil {
		d.goModCacheVolume = dag.CacheVolume("go-mod-cache")
	}

	return d.goModCacheVolume
}

func (d *JumppadCI) hasError() bool {
	return d.lastError != nil
}

func (d *JumppadCI) setArchLocalMachine(ctx context.Context) {
	// get the architecture of the current machine
	platform, err := dag.DefaultPlatform(ctx)
	if err != nil {
		panic(err)
	}

	arch := strings.Split(string(platform), "/")[1]
	os := strings.Split(string(platform), "/")[0]

	fmt.Println("Set build add for arch:", arch)

	oses = []string{os}
	arches = []string{arch}

	outputArch := arch
	if outputArch == "amd64" {
		outputArch = "x86_64"
	}

	// only change notorize if we are on darwin
	if os == "darwin" {
		filename := strings.Replace("jumppad_%%VERSION%%_darwin_%%ARCH%%.zip", "%%ARCH%%", outputArch, 1)
		notorize = []Archive{
			{Path: filename, Type: "zip", Output: filename},
		}

		archives = []Archive{
			{Path: fmt.Sprintf("/build/darwin/%s/jumppad.exe", arch), Type: "zip", Output: filename},
		}
	}

	if os == "linux" {
		filename := strings.Replace("jumppad_%%VERSION%%_darwin_%%ARCH%%.tar.gz", "%%ARCH%%", outputArch, 1)
		filenameDeb := strings.Replace("jumppad_%%VERSION%%_darwin_%%ARCH%%.deb", "%%ARCH%%", outputArch, 1)

		archives = []Archive{
			{Path: fmt.Sprintf("/build/linux/%s/jumppad", arch), Type: "targz", Output: filename},
			{Path: fmt.Sprintf("/pkg/linux/%s/jumppad.deb", arch), Type: "copy", Output: filenameDeb},
		}
	}
}

func startTestWorker(ctx context.Context, worker int, cli *Client, jumppad *File, src *Directory, arch string, jobs <-chan job, errors chan<- error) {
	for j := range jobs {
		wd := strings.TrimPrefix(j.workingDirectory, "/")
		//pl := cli.Pipeline("functional-test-" + wd + "-" + j.runtime)
		fmt.Println("Running test", worker, wd, j.runtime)

		// unique cache for the worker and the runtime, otherwise the cache will be shared
		// between the workers and when running in parallel this causes concurrency issues
		cache_name := fmt.Sprintf("%d_%s", worker, j.runtime)

		//time.Sleep(1 * time.Second)
		_, err := cli.Jumppad().
			TestBlueprintWithBinary(
				ctx,
				src,
				jumppad,
				JumppadTestBlueprintWithBinaryOpts{WorkingDirectory: j.workingDirectory, Architecture: arch, Runtime: j.runtime, Cache: cache_name},
			)

		errors <- err
	}
}
