package cmd

import (
	"strings"

	"github.com/jumppad-labs/jumppad/cmd/changelog"
	"github.com/spf13/cobra"
)

var changelogCmd = &cobra.Command{
	Use:   "changelog",
	Short: "Show the changelog",
	Long:  `Show the changelog`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cl := &changelog.Changelog{}

		// replace """ with ``` in changelog
		changes = strings.ReplaceAll(changes, `"""`, "```")

		err := cl.Show(changes, changesVersion, true)
		if err != nil {
			showErr(err)
		}
	},
}
