package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:                   "uninstall",
	Short:                 "Uninstall Shipyard",
	Long:                  `Uninstall Shipyard`,
	DisableFlagsInUseLine: true,
	Args:                  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// remove the config
		fmt.Println("Removing Shipyard configuration from", utils.ShipyardHome())
		err := os.RemoveAll(utils.ShipyardHome())
		if err != nil {
			fmt.Println("Error: Unable to remove Shipyard configuration", err)
			os.Exit(1)
		}

		// remove the binary
		ep, _ := os.Executable()
		cf, err := filepath.Abs(ep)
		if err != nil {
			fmt.Println("Error: Unable to remove Shipyard application", err)
			os.Exit(1)
		}
		fmt.Println("Removing Shipyard application from", cf)
		err = os.Remove(cf)
		if err != nil {
			fmt.Println("Error: Unable to remove Shipyard application", err)
			os.Exit(1)
		}

		fmt.Println("")
		fmt.Println("Shipyard successfully uninstalled")
	},
}
