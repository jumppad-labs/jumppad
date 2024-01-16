package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/jumppad/constants"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/spf13/cobra"
)

var taintCmd = &cobra.Command{
	Use:   "taint [resource]",
	Short: "Taint a resource e.g. 'jumppad taint container.test'",
	Long: `Taint a resource and mark is to be re-created on the next Apply
	Example use to remove a container named test
	jumppad taint resource.container.test
	`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			fmt.Println("The resource to taint must be specified as an argument")
			os.Exit(1)
		}

		cfg, err := config.LoadState()
		if err != nil {
			fmt.Println("Unable to load statefile, do you have a running blueprint?")
			os.Exit(1)
		}

		r, err := cfg.FindResource(args[0])
		if err != nil || r == nil {
			fmt.Println("Unable to locate resource in the state", args[0])
			os.Exit(1)
		}

		r.Metadata().ResourceProperties[constants.PropertyStatus] = constants.StatusTainted

		d, err := cfg.ToJSON()
		if err != nil {
			fmt.Println("Unable to save state", err)
			os.Exit(1)
		}

		ioutil.WriteFile(utils.StatePath(), d, os.ModePerm)
	},
}
