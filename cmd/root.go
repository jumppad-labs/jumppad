package cmd

import (
	"fmt"
	"os"

	"github.com/hashicorp/go-hclog"
	gvm "github.com/shipyard-run/version-manager"

	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/shipyard"
	"github.com/jumppad-labs/jumppad/pkg/utils"

	"github.com/spf13/cobra"
)

var configFile = ""

var rootCmd = &cobra.Command{
	Use:   "jumppad",
	Short: "Modern cloud native development environments",
	Long:  `Jumppad is a tool that helps you create and run development, demo, and tutorial environments`,
}

var engine shipyard.Engine
var logger hclog.Logger
var engineClients *clients.Clients

var version string // set by build process
var date string    // set by build process
var commit string  // set by build process

func init() {

	var vm gvm.Versions

	// setup dependencies
	logger = createLogger()
	engine, vm = createEngine(logger)
	engineClients = engine.GetClients()

	//cobra.OnInitialize(configure)

	//rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is $HOME/.shipyard/config)")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(outputCmd)
	rootCmd.AddCommand(newEnvCmd(engine))
	rootCmd.AddCommand(newRunCmd(engine, engineClients.Getter, engineClients.HTTP, engineClients.Browser, vm, engineClients.Connector, logger))
	rootCmd.AddCommand(newTestCmd(engine, engineClients.Getter, engineClients.HTTP, engineClients.Browser, logger))
	rootCmd.AddCommand(pauseCmd)
	rootCmd.AddCommand(resumeCmd)
	rootCmd.AddCommand(newGetCmd(engineClients.Getter))
	rootCmd.AddCommand(newDestroyCmd(engineClients.Connector))
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(newPurgeCmd(engineClients.Docker, engineClients.ImageLog, logger))
	rootCmd.AddCommand(taintCmd)
	rootCmd.AddCommand(newExecCmd(engineClients.ContainerTasks))
	rootCmd.AddCommand(newVersionCmd(vm))
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(newPushCmd(engineClients.ContainerTasks, engineClients.Kubernetes, engineClients.HTTP, engineClients.Nomad, logger))
	rootCmd.AddCommand(newLogCmd(engine, engineClients.Docker, os.Stdout, os.Stderr), completionCmd)

	// add the server commands
	rootCmd.AddCommand(connectorCmd)
	connectorCmd.AddCommand(newConnectorRunCommand())
	connectorCmd.AddCommand(connectorStopCmd)
	connectorCmd.AddCommand(newConnectorCertCmd())
}

func createEngine(l hclog.Logger) (shipyard.Engine, gvm.Versions) {
	engine, err := shipyard.New(l)
	if err != nil {
		panic(err)
	}

	o := gvm.Options{
		Organization: "jumppad-labs",
		Repo:         "jumppad",
		ReleasesPath: utils.GetReleasesFolder(),
	}

	o.AssetNameFunc = func(version, goos, goarch string) string {
		// No idea why we set the release architecture for the binary like this
		if goarch == "amd64" {
			goarch = "x86_64"
		}

		switch goos {
		case "linux":
			return fmt.Sprintf("jumppad_%s_%s_%s.tar.gz", version, goos, goarch)
		case "darwin":
			return fmt.Sprintf("jumppad_%s_%s_%s.zip", version, goos, goarch)
		case "windows":
			return fmt.Sprintf("jumppad_%s_%s_%s.zip", version, goos, goarch)
		}

		return ""
	}

	o.ExeNameFunc = func(version, goos, goarch string) string {
		if goos == "windows" {
			return "jumppad.exe"
		}

		return "jumppad"
	}

	vm := gvm.New(o)

	return engine, vm
}

func createLogger() hclog.Logger {

	opts := &hclog.LoggerOptions{Color: hclog.AutoColor}

	// set the log level
	if lev := os.Getenv("LOG_LEVEL"); lev != "" {
		opts.Level = hclog.LevelFromString(lev)
	}

	return hclog.New(opts)
}

// Execute the root command
func Execute(v, c, d string) error {
	version = v
	commit = c
	date = d

	err := rootCmd.Execute()

	if err != nil {
		fmt.Println(discordHelp)
	}

	return err
}

var discordHelp = `
### For help and support join our community on Discord: https://discord.gg/ZuEFPJU69D ###
`
