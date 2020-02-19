package cmd

import (
	"fmt"
	"os"

	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"
)

var taintCmd = &cobra.Command{
	Use:   "taint [type].[name]",
	Short: "Taint a resource e.g. 'shipyard taint container test'",
	Long: `Taint a resouce and mark is to be re-created on the next Apply
	Example use to remove a container named test
	shipyard taint container.test	
	`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			fmt.Println("The resource to taint must be specified as an argument")
			os.Exit(1)
		}

		c := config.New()
		err := c.FromJSON(utils.StatePath())
		if err != nil {
			fmt.Println("Unable to load state", err)
			os.Exit(1)
		}

		r, err := c.FindResource(args[0])
		if err != nil || r == nil {
			fmt.Println("Unable to locate resource in the state", args[0])
			os.Exit(1)
		}

		r.Info().Status = config.PendingModification

		err = c.ToJSON(utils.StatePath())
		if err != nil {
			fmt.Println("Unable to save state", err)
			os.Exit(1)
		}
	},
}
