package cmd

import (
	"fmt"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/shipyard"
	"github.com/shipyard-run/shipyard/pkg/utils"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configFile = ""

var rootCmd = &cobra.Command{
	Use:   "yard",
	Short: "Modern cloud native development environments",
	Long:  `Shipyard is a tool that helps you create and run demo and tutorial environments`,
}

var engine shipyard.Engine
var logger hclog.Logger
var engineClients *shipyard.Clients

var version string

func init() {
	// create the shipyard home
	os.MkdirAll(utils.ShipyardHome(), os.FileMode(0755))

	// setup dependencies
	var err error
	logger = createLogger()
	engine, err = shipyard.New(logger)
	if err != nil {
		panic(err)
	}

	engineClients, err = shipyard.GenerateClients(logger)
	if err != nil {
		panic(err)
	}

	cobra.OnInitialize(configure)

	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is $HOME/.shipyard/config)")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(newRunCmd(engine, engineClients.Blueprints, engineClients.HTTP, engineClients.Browser, logger))
	rootCmd.AddCommand(pauseCmd)
	rootCmd.AddCommand(resumeCmd)
	rootCmd.AddCommand(newGetCmd(engineClients.Blueprints))
	rootCmd.AddCommand(destroyCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(taintCmd)
	rootCmd.AddCommand(newExecCmd(engineClients.ContainerTasks))
	rootCmd.AddCommand(versionCmd)
	//rootCmd.AddCommand(exposeCmd)
	//rootCmd.AddCommand(containerCmd)
	//rootCmd.AddCommand(codeCmd)
	//rootCmd.AddCommand(docsCmd)
	//rootCmd.AddCommand(toolsCmd)
	//rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(pushCmd)
}

func configure() {
	if configFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(configFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
		}

		// Search config in home directory with name ".shipyard".
		viper.AddConfigPath(home)
		viper.SetConfigName(".shipyard/config")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

// Execute the root command
func Execute(v string) error {
	version = v
	return rootCmd.Execute()
}
