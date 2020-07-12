package cmd

import (
	"fmt"
	"path"

	"github.com/hashicorp/go-hclog"
	gvm "github.com/nicholasjackson/version-manager"
	"github.com/shipyard-run/shipyard/pkg/shipyard"
	"github.com/shipyard-run/shipyard/pkg/utils"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	// setup dependencies
	var err error
	logger = createLogger()
	engine, err = shipyard.New(logger)
	if err != nil {
		panic(err)
	}

	engineClients = engine.GetClients()

	o := gvm.Options{
		Organization: "shipyard-run",
		Repo:         "shipyard",
		ReleasesPath: path.Join(utils.ShipyardHome(), "releases"),
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

	cobra.OnInitialize(configure)

	//rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is $HOME/.shipyard/config)")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(checkCmd)
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
