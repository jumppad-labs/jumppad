package cmd

import (
	"fmt"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var config = ""

var rootCmd = &cobra.Command{
	Use:   "yard",
	Short: "Yard short description",
	Long: `Yard long description`,
	Run: func(cmd *cobra.Command, args []string) {
	  // Do Stuff Here
	},
}
  
func init() {
	cobra.OnInitialize(configure)

	rootCmd.PersistentFlags().StringVar(&config, "config", "", "config file (default is $HOME/.shipyard)")
}

func configure() {
	if config != "" {
		// Use config file from the flag.
		viper.SetConfigFile(config)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
		}

		// Search config in home directory with name ".shipyard".
		viper.AddConfigPath(home)
		viper.SetConfigName(".shipyard")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

  // Execute the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}