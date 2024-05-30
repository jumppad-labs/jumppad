package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/jumppad-labs/jumppad/cmd/changelog"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/jumppad"
	"github.com/jumppad-labs/jumppad/pkg/utils"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "jumppad",
	Short: "Modern cloud native development environments",
	Long:  `Jumppad is a tool that helps you create and run development, demo, and tutorial environments`,
}

var version string // set by build process
var date string    // set by build process
var commit string  // set by build process

// globalFlags are flags that are set for every command
func globalFlags() *pflag.FlagSet {
	flags := pflag.NewFlagSet("global", pflag.ContinueOnError)
	flags.Bool("non-interactive", false, "Run in non-interactive mode")

	return flags
}

func createEngine(l logger.Logger, c *clients.Clients, defaultRegistry string, registryCredentials map[string]string) (jumppad.Engine, error) {
	providers := config.NewProviders(c)

	engine, err := jumppad.New(providers, l, defaultRegistry, registryCredentials)
	if err != nil {
		return nil, err
	}

	return engine, nil
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

	viper.AddConfigPath(utils.JumppadHome())
	viper.SetConfigName("config")
	err := viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			l.Debug("No config file found, using defaults")
		} else {
			return err
		}
	}

	defaultRegistry := jumppad.GetDefaultRegistry()
	registryCredentials := jumppad.GetRegistryCredentials()

	engineClients, _ := clients.GenerateClients(l)

	engine, _ := createEngine(l, engineClients, defaultRegistry, registryCredentials)

	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(outputCmd)
	rootCmd.AddCommand(newDevCmd())
	rootCmd.AddCommand(newEnvCmd(engine))
	rootCmd.AddCommand(newRunCmd(engine, engineClients.ContainerTasks, engineClients.Getter, engineClients.HTTP, engineClients.System, engineClients.Connector, l))
	rootCmd.AddCommand(newTestCmd())
	rootCmd.AddCommand(newDestroyCmd(engineClients.Connector, l))
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(newPurgeCmd(engineClients.Docker, engineClients.ImageLog, l))
	rootCmd.AddCommand(taintCmd)
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(newPushCmd(engineClients.ContainerTasks, engineClients.Kubernetes, engineClients.HTTP, engineClients.Nomad, l))
	rootCmd.AddCommand(newLogCmd(engine, engineClients.Docker, os.Stdout, os.Stderr), completionCmd)
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

	err = rootCmd.Execute()

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
