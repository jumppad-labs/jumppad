package cmd

import (
	"strings"

	gvm "github.com/shipyard-run/version-manager"
	"github.com/spf13/cobra"
)

func newVersionInstallCmd(vm gvm.Versions) *cobra.Command {
	return &cobra.Command{
		Use:   "install [version]",
		Short: "Install a jumppad version",
		Long:  "Install a jumppad version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ver := strings.TrimSpace(args[0])
			cmd.Println("Installing version", ver)

			tag, rl, err := vm.GetLatestReleaseURL(ver)
			if err != nil {
				return err
			}

			cmd.Println("Downloading", rl)
			_, err = vm.DownloadRelease(tag, rl)
			if err != nil {
				return err
			}

			return nil
		},
	}
}
