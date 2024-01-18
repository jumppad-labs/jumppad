package main

import (
	"fmt"
)

var oses = []string{"linux", "darwin", "windows"}
var arches = []string{"amd64", "arm64"}

func New() *Jumppad {
	return &Jumppad{}
}

type Jumppad struct {
	lastError error
}

func (d *Jumppad) All(src *Directory) (*Directory, error) {
	output, _ := d.Build(src)

	// package the build outputs
	return d.Package(output)
}

func (d *Jumppad) Build(src *Directory) (*Directory, error) {
	if d.hasError() {
		return nil, d.lastError
	}

	src = src.WithoutDirectory(".dagger")

	fmt.Println("Building...")

	// create empty directory to put build outputs
	outputs := dag.Directory()

	// get `golang` image
	golang := dag.Container().From("golang:latest")

	// mount cloned repository into `golang` image
	golang = golang.WithDirectory("/src", src).WithWorkdir("/src")

	for _, goos := range oses {
		for _, goarch := range arches {
			fmt.Println("Build for", goos, goarch, "...")

			// create a directory for each os and arch
			path := fmt.Sprintf("build/%s/%s/", goos, goarch)

			// set GOARCH and GOOS in the build environment
			build := golang.
				WithEnvVariable("CGO_ENABLED", "0").
				WithEnvVariable("GOOS", goos).
				WithEnvVariable("GOARCH", goarch).
				WithExec([]string{"go", "build", "-o", path})

			// get reference to build output directory in container
			outputs = outputs.WithDirectory(path, build.Directory(path))
		}
	}

	return outputs, nil
}

func (d *Jumppad) Package(binaries *Directory) (*Directory, error) {
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

func (d *Jumppad) hasError() bool {
	return d.lastError != nil
}
