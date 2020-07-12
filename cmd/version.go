package cmd

import (
	"fmt"
	"path"

	gvm "github.com/nicholasjackson/version-manager"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:           "version",
	Short:         "Shipyard version manager commands",
	Long:          "Shipyard version manager commands",
	Args:          cobra.NoArgs,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Println("Current Version:", version)
		cmd.Println("")

		return fmt.Errorf("")
	},
}

func init() {
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

	g := gvm.New(o)

	// Add the sub commands
	versionCmd.AddCommand(newVersionListCmd(g))
	versionCmd.AddCommand(newVersionInstallCmd(g))
}
