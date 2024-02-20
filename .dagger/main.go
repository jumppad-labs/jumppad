package main

import (
	"context"
	"fmt"
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
	lastError         error
	goCacheVolume     *CacheVolume
	dockerCacheVolume *CacheVolume
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

	var err error
	var output *Directory

	// remove the build output directory from the source
	src = src.
		WithoutDirectory(".dagger").
		WithoutDirectory("build-output")

	// if we have a github token, get the version from the associated PR label
	if githubToken != nil {
		version, sha, err = d.getVersion(ctx, githubToken, src)
	}

	log.Info("Building version", "semver", version, "sha", sha)

	// run the unit tests
	d.UnitTest(ctx, src, !quick)

	// build the applications
	output, err = d.Build(ctx, src)

	// package the build outputs
	output, err = d.Package(ctx, output, version)

	// create the archives
	output, err = d.Archive(ctx, output, version)

	if notorizeCert != nil && notorizeCertPassword != nil && notorizeKey != nil && notorizeId != "" && notorizeIssuer != "" {
		output, err = d.SignAndNotorize(ctx, version, output, notorizeCert, notorizeCertPassword, notorizeKey, notorizeId, notorizeIssuer)
	}

	return output, err
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

func (d *JumppadCI) Build(ctx context.Context, src *Directory) (*Directory, error) {
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
		WithMountedCache("/go/pkg/mod", d.goCache())

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
				WithExec([]string{"go", "build", "-o", path}).
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

func (d *JumppadCI) UnitTest(ctx context.Context, src *Directory, withRace bool) error {
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
		WithMountedCache("/go/pkg/mod", d.goCache()).
		WithWorkdir("/src").
		WithExec([]string{"go", "test", "-v", raceFlag, "./..."})

	_, err := golang.Sync(ctx)
	if err != nil {
		d.lastError = err
	}

	return err
}

func (d *JumppadCI) Package(ctx context.Context, binaries *Directory, version string) (*Directory, error) {
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
func (d *JumppadCI) Archive(ctx context.Context, binaries *Directory, version string) (*Directory, error) {
	if d.hasError() {
		return nil, d.lastError
	}

	cli := dag.Pipeline("archive")
	out := cli.Directory()

	zipContainer := cli.Container().
		From("alpine:latest").
		WithExec([]string{"apk", "add", "zip"})

	for _, a := range archives {
		outPath := strings.ReplaceAll(a.Output, "%%VERSION%%", version)
		switch a.Type {
		case "zip":
			// create a zip archive
			zip := zipContainer.
				WithMountedFile("/jumppad", binaries.File(a.Path)).
				WithExec([]string{"zip", "-r", outPath, "/jumppad"})

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
	}

	return out, nil
}

var notorize = []Archive{
	{Path: "/jumppad_%%VERSION%%_darwin_x86_64.zip", Type: "zip", Output: "/jumppad_%%VERSION%%_darwin_x86_64.zip"},
	{Path: "/jumppad_%%VERSION%%_darwin_arm64.zip", Type: "zip", Output: "/jumppad_%%VERSION%%_darwin_arm64.zip"},
}

// SignAndNotorize signs and notorizes the osx binaries using the Apple notary service
func (d JumppadCI) SignAndNotorize(ctx context.Context, version string, archives *Directory, cert *File, password *Secret, key *File, keyId, keyIssuer string) (*Directory, error) {
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

func (d *JumppadCI) Release(ctx context.Context, src *Directory, archives *Directory, githubToken *Secret) (string, error) {
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

func (d *JumppadCI) WithGoCache(cache *CacheVolume) *JumppadCI {
	d.goCacheVolume = cache
	return d
}

func (d *JumppadCI) goCache() *CacheVolume {
	if d.goCacheVolume == nil {
		d.goCacheVolume = dag.CacheVolume("go-cache")
	}

	return d.goCacheVolume
}

func (d *JumppadCI) dockerCache() *CacheVolume {
	if d.dockerCacheVolume == nil {
		d.dockerCacheVolume = dag.CacheVolume("docker-cache")
	}

	return d.dockerCacheVolume
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

var functionalTests = []string{
	//	"/examples/build",
	//	"/examples/certificates",
	//	"/examples/container",
	//	"/examples/docs",
	//	"/examples/exec",
	"/examples/multiple_k3s_clusters",
	// "/examples/nomad",
	// "/examples/single_file",
	// "/examples/single_k3s_cluster",
	// "/examples/terraform",
}

func (d *JumppadCI) FunctionalTestAll(ctx context.Context, jumppad *File, src *Directory, architecture, runtime string) error {
	if d.hasError() {
		return d.lastError
	}

	for _, ft := range functionalTests {
		testDir := src.Directory(ft)

		_, err := dag.Jumppad().
			WithCache(d.dockerCache()).
			TestBlueprintWithBinary(
				ctx,
				testDir,
				jumppad,
				JumppadTestBlueprintWithBinaryOpts{Architecture: architecture, Runtime: runtime},
			)

		if err != nil {
			d.lastError = err
			return err
		}
	}

	return nil
}
