package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jumppad-labs/hclconfig"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/build"
	cp "github.com/otiai10/copy"
	"github.com/spf13/cobra"
	"golang.org/x/mod/modfile"
)

type Plugin struct {
	Alias   string `hcl:"alias,label"`    // alias of the plugin
	Source  string `hcl:"source"`         // git repository containing the plugin
	Local   string `hcl:"local,optional"` // git repository containing the plugin
	Version string `hcl:"version"`        // git ref
}

type Jumppad struct {
	types.ResourceBase `hcl:",remain"`

	Version string   `hcl:"version,optional"` // version of the jumppad to build, if not set, main will use the latest
	Plugins []Plugin `hcl:"plugin,block"`
}

var pluginCmd = &cobra.Command{
	Use:   "build",
	Short: "Checks the system to ensure required dependencies are installed",
	Long:  `Checks the system to ensure required dependencies are installed`,
	Args:  cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {

		opts := hclconfig.DefaultOptions()
		p := hclconfig.NewParser(opts)

		p.RegisterType("jumppad", &Jumppad{})

		file, err := filepath.Abs(args[0])
		if err != nil {
			panic(err)
		}

		// Parse the config file
		cfg, err := p.ParseFile(file)
		if err != nil {
			panic(err)
		}

		// Get the parsed config
		jumppads, err := cfg.FindResourcesByType("jumppad")
		if err != nil {
			panic(err)
		}

		l := createLogger()
		engineClients, _ := clients.GenerateClients(l)

		// create a temp output folder
		tmp := os.TempDir()
		output := filepath.Join(tmp, "jumppad_build")

		os.RemoveAll(output)

		os.MkdirAll(output, 0755)

		src := filepath.Join(output, "src")

		// download the source
		engineClients.Getter.Get("github.com/jumppad-labs/jumppad?ref=main", src)

		fmt.Println("Downloaded source to", src)
		d, err := os.ReadFile(filepath.Join(src, "go.mod"))
		if err != nil {
			panic(err)
		}

		// parse the go mod
		gomod, err := modfile.Parse("go.mod", d, nil)
		if err != nil {
			panic(err)
		}
		fmt.Println(gomod.Go.Version)

		for _, jumppad := range jumppads {
			j := jumppad.(*Jumppad)
			for _, plugin := range j.Plugins {
				fmt.Println(plugin.Alias, plugin.Source, plugin.Version)
				gomod.AddRequire(plugin.Source, plugin.Version)

				if plugin.Local != "" {
					// copy the local plugin source to the output/local folder
					// so we can add it to the docker build context
					local := filepath.Join(output, "local", plugin.Alias)
					os.MkdirAll(local, 0755)
					cp.Copy(plugin.Local, local)

					gomod.AddReplace(plugin.Source, "", filepath.Join("/local", plugin.Alias), "")
				}
			}
		}

		// write the go mod
		generated := filepath.Join(output, "generated")
		os.MkdirAll(generated, 0755)

		d, err = gomod.Format()
		if err != nil {
			panic(err)
		}

		err = os.WriteFile(filepath.Join(generated, "go.mod"), d, 0644)
		if err != nil {
			panic(err)
		}

		// add the custom init for the plugins
		init := strings.Builder{}
		init.WriteString("package jumppad\n\n")

		for _, jumppad := range jumppads {
			j := jumppad.(*Jumppad)
			for _, plugin := range j.Plugins {
				init.WriteString(fmt.Sprintf("import %s \"%s\"\n", plugin.Alias, plugin.Source))
			}
		}

		init.WriteString("\n")
		init.WriteString("func init() {\n")

		for _, jumppad := range jumppads {
			j := jumppad.(*Jumppad)
			for _, plugin := range j.Plugins {
				init.WriteString(fmt.Sprintf("  %s.Register(PluginRegisterResource,PluginLoadState)\n", plugin.Alias))
			}
		}

		init.WriteString("}\n")

		err = os.WriteFile(filepath.Join(src, "pkg", "jumppad", "plugin.go"), []byte(init.String()), 0644)
		if err != nil {
			panic(err)
		}

		// add the dockerfile
		buildSrc := filepath.Join(output, "build")
		os.MkdirAll(buildSrc, 0755)

		err = os.WriteFile(filepath.Join(buildSrc, "Dockerfile.build"), []byte(dockerfile), 0644)
		if err != nil {
			panic(err)
		}

		bin := filepath.Join(output, "bin")
		os.MkdirAll(bin, 0755)

		prov := &build.Provider{}
		prov.Init(&build.Build{
			ResourceBase: types.ResourceBase{
				Meta: types.Meta{
					Name: "jumppad",
				},
			},
			Container: build.BuildContainer{
				DockerFile: filepath.Join("build", "Dockerfile.build"),
				Context:    output,
				Args: map[string]string{
					"ARCH": runtime.GOARCH,
					"OS":   runtime.GOOS,
				},
			},
			Outputs: []build.Output{
				build.Output{
					Source:      "/src/bin/jumppad",
					Destination: filepath.Join(bin, "jumppad"),
				},
			},
		}, l)

		err = prov.Create()
		if err != nil {
			panic(err)
		}
	},
}

var dockerfile = `
FROM "golang:1.21" AS builder

ARG ARCH=amd64
ARG OS=linux

WORKDIR /src

COPY ./src /src
COPY ./generated/go.mod /src/go.mod
COPY ./local /local

RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=${OS} GOARCH=${ARCH} go build -ldflags "-X main.version=custom" -o bin/jumppad main.go
`
