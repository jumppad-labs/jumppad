package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

var uninstallCmd = &cobra.Command{
	Use:                   "uninstall",
	Short:                 "Uninstall shipyard",
	Long:                  `Uninstall shipyard`,
	DisableFlagsInUseLine: true,
	Args:                  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// remove the config
		fmt.Println("Removing Shipyard configuration from", ShipyardHome())
		err := os.RemoveAll(ShipyardHome())
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
