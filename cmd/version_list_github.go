package cmd

import (
	gvm "github.com/nicholasjackson/version-manager"
	"github.com/spf13/cobra"
)

func newVersionListCmd(vm gvm.Versions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the available Shipyard versions",
		Long:  "List the available Shipyard versions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("Current Version:", version)
			cmd.Println("")

			r, err := vm.ListInstalledVersions("")
			if err != nil {
				return err
			}

			cmd.Println("Installed Versions:")
			cmd.Println("")
			cmd.Println("Version  | Url")
			cmd.Println("________ | ______________________________________________________________")

			// sort the keys
			keys := vm.SortMapKeys(r, true)

			for _, k := range keys {
				cmd.Printf("%-8s | %s\n", k, r[k])
			}

			cmd.Println("")
			cmd.Println("")

			cmd.Println("Available Versions:")
			cmd.Println("")
			cmd.Println("Version  | Url")
			cmd.Println("________ | ______________________________________________________________")

			r, err = vm.ListReleases("")
			if err != nil {
				return err
			}

			// sort the keys
			keys = vm.SortMapKeys(r, true)

			for _, k := range keys {
				cmd.Printf("%-8s | %s\n", k, r[k])
			}

			return nil
		},
	}
}
