package cmd

import (
	"fmt"
	"os"

	"github.com/shipyard-run/shipyard/pkg/shipyard"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configFile = ""

var rootCmd = &cobra.Command{
	Use:   "yard",
	Short: "Modern cloud native development environments",
	Long:  `Shipyard is a tool that helps you create and run demo and tutorial environments`,
	Run: func(cmd *cobra.Command, args []string) {
		// engine = shipyard.New()
	},
}

var engine *shipyard.Engine

func init() {
	cobra.OnInitialize(configure)

	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is $HOME/.shipyard/config)")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(destroyCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(exposeCmd)
	rootCmd.AddCommand(containerCmd)
	rootCmd.AddCommand(codeCmd)
	rootCmd.AddCommand(docsCmd)
	rootCmd.AddCommand(toolsCmd)
	rootCmd.AddCommand(upgradeCmd)
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
func Execute(version string) {
	// TODO implemnent a version subcommand
	fmt.Println("Shipyard version:", version)
	fmt.Println("")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
