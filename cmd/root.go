package cmd

import (
	"fmt"
	"os"

	gvm "github.com/shipyard-run/version-manager"

	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/jumppad"
	"github.com/jumppad-labs/jumppad/pkg/utils"

	"github.com/spf13/cobra"
)

var configFile = ""

var rootCmd = &cobra.Command{
	Use:   "jumppad",
	Short: "Modern cloud native development environments",
	Long:  `Jumppad is a tool that helps you create and run development, demo, and tutorial environments`,
}

var engine jumppad.Engine
var l logger.Logger
var engineClients *clients.Clients

var version string // set by build process
var date string    // set by build process
var commit string  // set by build process

func init() {

	var vm gvm.Versions

	// setup dependencies
	l = createLogger()
	engine, engineClients, vm = createEngine(l)

	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(outputCmd)
	rootCmd.AddCommand(newDevCmd())
	rootCmd.AddCommand(newEnvCmd(engine))
	rootCmd.AddCommand(newRunCmd(engine, engineClients.ContainerTasks, engineClients.Getter, engineClients.HTTP, engineClients.Browser, vm, engineClients.Connector, l))
	rootCmd.AddCommand(newTestCmd())
	rootCmd.AddCommand(newDestroyCmd(engineClients.Connector))
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(newPurgeCmd(engineClients.Docker, engineClients.ImageLog, l))
	rootCmd.AddCommand(taintCmd)
	rootCmd.AddCommand(newVersionCmd(vm))
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(newPushCmd(engineClients.ContainerTasks, engineClients.Kubernetes, engineClients.HTTP, engineClients.Nomad, l))
	rootCmd.AddCommand(newLogCmd(engine, engineClients.Docker, os.Stdout, os.Stderr), completionCmd)

	// add the server commands
	rootCmd.AddCommand(connectorCmd)
	connectorCmd.AddCommand(newConnectorRunCommand())
	connectorCmd.AddCommand(connectorStopCmd)
	connectorCmd.AddCommand(newConnectorCertCmd())

	// add the generate command
	rootCmd.AddCommand(generateCmd)
	generateCmd.AddCommand(newGenerateReadmeCommand(engine))
}

func createEngine(l logger.Logger) (jumppad.Engine, *clients.Clients, gvm.Versions) {
	engineClients, err := clients.GenerateClients(l)
	if err != nil {
		return nil, nil, nil
	}

	providers := config.NewProviders(engineClients)

	engine, err := jumppad.New(providers, l)
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

	return engine, engineClients, vm
}

func createLogger() logger.Logger {
	// set the log level
	if lev := os.Getenv("LOG_LEVEL"); lev != "" {
		return logger.NewLogger(os.Stdout, lev)
	}

	return logger.NewLogger(os.Stdout, logger.LogLevelInfo)
}

// Execute the root command
func Execute(v, c, d string) error {
	version = v
	commit = c
	date = d

	rootCmd.SilenceErrors = true

	err := rootCmd.Execute()

	if err != nil {
		fmt.Println("")
		fmt.Println(err)
		fmt.Println(discordHelp)
	}

	return err
}

var discordHelp = `
### For help and support join our community on Discord: https://discord.gg/ZuEFPJU69D ###
`
