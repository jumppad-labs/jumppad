package cmd

import (
	"fmt"

	"github.com/hashicorp/go-hclog"
	gvm "github.com/nicholasjackson/version-manager"
	"github.com/shipyard-run/shipyard/pkg/shipyard"
	"github.com/shipyard-run/shipyard/pkg/utils"

	"github.com/spf13/cobra"
)

var configFile = ""

var rootCmd = &cobra.Command{
	Use:   "shipyard",
	Short: "Modern cloud native development environments",
	Long:  `Shipyard is a tool that helps you create and run development, demo, and tutorial environments`,
}

var engine shipyard.Engine
var logger hclog.Logger
var engineClients *shipyard.Clients

var version string

func init() {
	var vm gvm.Versions

	// setup dependencies
	logger = createLogger()
	engine, vm = createEngine(logger)
	engineClients := engine.GetClients()

	//cobra.OnInitialize(configure)

	//rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is $HOME/.shipyard/config)")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(outputCmd)
	rootCmd.AddCommand(newEnvCmd(engine))
	rootCmd.AddCommand(newRunCmd(engine, engineClients.Getter, engineClients.HTTP, engineClients.Browser, vm, logger))
	rootCmd.AddCommand(newTestCmd(engine, engineClients.Getter, engineClients.HTTP, engineClients.Browser, logger))
	rootCmd.AddCommand(pauseCmd)
	rootCmd.AddCommand(resumeCmd)
	rootCmd.AddCommand(newGetCmd(engineClients.Getter))
	rootCmd.AddCommand(destroyCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(newPurgeCmd(engineClients.Docker, engineClients.ImageLog, logger))
	rootCmd.AddCommand(taintCmd)
	rootCmd.AddCommand(newExecCmd(engineClients.ContainerTasks))
	rootCmd.AddCommand(newVersionCmd(vm))
	//rootCmd.AddCommand(exposeCmd)
	//rootCmd.AddCommand(containerCmd)
	//rootCmd.AddCommand(codeCmd)
	//rootCmd.AddCommand(docsCmd)
	//rootCmd.AddCommand(toolsCmd)
	//rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(newPushCmd(engineClients.ContainerTasks, engineClients.Kubernetes, engineClients.HTTP, engineClients.Nomad, logger))
}

func createEngine(l hclog.Logger) (shipyard.Engine, gvm.Versions) {
	engine, err := shipyard.New(l)
	if err != nil {
		panic(err)
	}

	o := gvm.Options{
		Organization: "shipyard-run",
		Repo:         "shipyard",
		ReleasesPath: utils.GetReleasesFolder(),
	}

	o.AssetNameFunc = func(version, goos, goarch string) string {
		// No idea why we set the release architecture for the binary like this
		if goarch == "amd64" {
			goarch = "x86_64"
		}

		// zip is used on windows as tar is not available by default
		switch goos {
		case "linux":
			return fmt.Sprintf("shipyard_%s_%s_%s.tar.gz", version, goos, goarch)
		case "darwin":
			return fmt.Sprintf("shipyard_%s_%s_%s.tar.gz", version, goos, goarch)
		case "windows":
			return fmt.Sprintf("shipyard_%s_%s_%s.zip", version, goos, goarch)
		}

		return ""
	}

	o.ExeNameFunc = func(version, goos, goarch string) string {
		if goos == "windows" {
			return "shipyard.exe"
		}

		return "shipyard"
	}

	vm := gvm.New(o)

	return engine, vm
}

// Execute the root command
func Execute(v string) error {
	version = v
	return rootCmd.Execute()
}
