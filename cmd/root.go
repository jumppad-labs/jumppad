package cmd

import (
	"fmt"
	"os"
	"strings"

	gvm "github.com/shipyard-run/version-manager"

	"github.com/jumppad-labs/jumppad/cmd/changelog"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/jumppad"
	"github.com/jumppad-labs/jumppad/pkg/utils"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "jumppad",
	Short: "Modern cloud native development environments",
	Long:  `Jumppad is a tool that helps you create and run development, demo, and tutorial environments`,
}

var version string //lint:ignore U1000 set at runtime
var date string    //lint:ignore U1000 set at runtime
var commit string  //lint:ignore U1000 set at runtime

func createEngine(l logger.Logger, c *clients.Clients) (jumppad.Engine, gvm.Versions, error) {
	providers := config.NewProviders(c)

	engine, err := jumppad.New(providers, l)
	if err != nil {
		return nil, nil, err
	}

	o := gvm.Options{
		Organization: "jumppad-labs",
		Repo:         "jumppad",
		ReleasesPath: utils.ReleasesFolder(),
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

	return engine, vm, nil
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

	// setup dependencies
	l := createLogger()

	engineClients, _ := clients.GenerateClients(l)

	engine, _, _ := createEngine(l, engineClients)

	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(outputCmd)
	rootCmd.AddCommand(newDevCmd())
	rootCmd.AddCommand(newEnvCmd())
	rootCmd.AddCommand(newRunCmd(engine, engineClients.ContainerTasks, engineClients.Getter, engineClients.HTTP, engineClients.System, engineClients.Connector, l))
	rootCmd.AddCommand(newTestCmd())
	rootCmd.AddCommand(newDestroyCmd(engineClients.Connector, l))
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(newPurgeCmd(engineClients.Docker, engineClients.ImageLog, l))
	rootCmd.AddCommand(taintCmd)
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(newPushCmd(engineClients.ContainerTasks, l))
	rootCmd.AddCommand(newLogCmd(engineClients.Docker, os.Stdout, os.Stderr), completionCmd)
	rootCmd.AddCommand(changelogCmd)

	// add the server commands
	rootCmd.AddCommand(connectorCmd)
	connectorCmd.AddCommand(newConnectorRunCommand())
	connectorCmd.AddCommand(connectorStopCmd)
	connectorCmd.AddCommand(newConnectorCertCmd())

	// add the generate command
	rootCmd.AddCommand(generateCmd)
	generateCmd.AddCommand(newGenerateReadmeCommand(engine))

	// add the plugin commands
	rootCmd.AddCommand(pluginCmd)

	rootCmd.SilenceErrors = true

	// set a pre run function to show the changelog
	rootCmd.PersistentFlags().Bool("non-interactive", false, "Run in non-interactive mode")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		ni, _ := cmd.Flags().GetBool("non-interactive")
		if ni {
			return nil
		}

		cl := &changelog.Changelog{}

		// replace """ with ``` in changelog
		changes = strings.ReplaceAll(changes, `"""`, "```")

		err := cl.Show(changes, changesVersion, false)
		if err != nil {
			showErr(err)
			return err
		}

		// Check the system to see if Docker is running and everything is installed
		s, err := engineClients.System.Preflight()
		if err != nil {
			fmt.Println("")
			fmt.Println("###### SYSTEM DIAGNOSTICS ######")
			fmt.Println(s)
			return err
		}

		return nil
	}

	err := rootCmd.Execute()

	if err != nil {
		showErr(err)
	}

	return err
}

func showErr(err error) {
	fmt.Println("")
	fmt.Println(err)
	fmt.Println(discordHelp)
}

var discordHelp = `
### For help and support join our community on Discord: https://discord.gg/ZuEFPJU69D ###
`
