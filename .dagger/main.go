package main

import (
	"context"
	"fmt"
	"strings"
)

var oses = []string{"linux", "darwin", "windows"}
var arches = []string{"amd64", "arm64"}

func New() *JumppadCI {
	return &JumppadCI{}
}

type JumppadCI struct {
	lastError         error
	goCacheVolume     *CacheVolume
	dockerCacheVolume *CacheVolume
}

func (d *JumppadCI) All(ctx context.Context, src *Directory) (*Directory, error) {
	src = src.
		WithoutDirectory(".dagger").
		WithoutDirectory(".git").
		WithoutDirectory("output")

	// get the architecture of the current machine
	platform, err := dag.DefaultPlatform(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get default architecture: %w", err)
	}
	arch := strings.Split(string(platform), "/")[1]

	fmt.Println("Build add for arch:", arch)

	// unit test
	d.UnitTest(ctx, src, true)

	// build for all achitectures and get the build outputs
	output, _ := d.Build(ctx, src)

	// package the build outputs
	return d.Package(ctx, output, "0.0.0")
}

func (d *JumppadCI) Quick(ctx context.Context, src *Directory) (*Directory, error) {
	src = src.
		WithoutDirectory(".dagger").
		WithoutDirectory(".git").
		WithoutDirectory("output")

	// get the architecture of the current machine
	platform, err := dag.DefaultPlatform(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get default architecture: %w", err)
	}
	arch := strings.Split(string(platform), "/")[1]

	fmt.Println("Build add for arch:", arch)

	oses = []string{"linux"}
	arches = []string{arch}

	// unit test
	d.UnitTest(ctx, src, false)

	// build for all achitectures and get the build outputs
	output, _ := d.Build(ctx, src)

	// package the build outputs
	return d.Package(ctx, output, "0.0.0")
}

func (d *JumppadCI) Build(ctx context.Context, src *Directory) (*Directory, error) {
	if d.hasError() {
		return nil, d.lastError
	}

	fmt.Println("Building...")

	// create empty directory to put build outputs
	outputs := dag.Directory()

	// get `golang` image
	golang := dag.Container().
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

	raceFlag := ""
	if withRace {
		raceFlag = "-race"
	}

	golang := dag.Container().
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

	archs := []string{"amd64", "arm64"}

	for _, a := range archs {
		// create a package directory including the binaries
		pkg := dag.Directory()
		pkg = pkg.WithFile("/bin/jumppad", binaries.File(fmt.Sprintf("/build/linux/%s/jumppad", a)))

		// create a debian package
		p := dag.Deb().Build(pkg, a, "jumppad", "0.0.1", "Nic Jackson", "Jumppad application")

		// add the debian package to the binaries directory
		binaries = binaries.WithFile(fmt.Sprintf("/pkg/linux/%s/jumppad.deb", a), p)
	}

	return binaries, nil
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
