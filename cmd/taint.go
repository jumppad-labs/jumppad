package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/shipyard-run/hclconfig"
	"github.com/shipyard-run/shipyard/pkg/shipyard/constants"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"
)

var taintCmd = &cobra.Command{
	Use:   "taint [type].[name]",
	Short: "Taint a resource e.g. 'shipyard taint container.test'",
	Long: `Taint a resource and mark is to be re-created on the next Apply
	Example use to remove a container named test
	shipyard taint container.test
	`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			fmt.Println("The resource to taint must be specified as an argument")
			os.Exit(1)
		}

		p := hclconfig.NewParser(hclconfig.DefaultOptions())
		d, err := ioutil.ReadFile(utils.StatePath())
		if err != nil {
			fmt.Printf("Unable to read state file")
			os.Exit(1)
		}

		cfg, err := p.UnmarshalJSON(d)
		if err != nil {
			fmt.Printf("Unable to unmarshal state file")
			os.Exit(1)
		}

		r, err := cfg.FindResource(args[0])
		if err != nil || r == nil {
			fmt.Println("Unable to locate resource in the state", args[0])
			os.Exit(1)
		}

		r.Metadata().Properties[constants.PropertyStatus] = constants.StatusTainted

		d, err = cfg.ToJSON()
		if err != nil {
			fmt.Println("Unable to save state", err)
			os.Exit(1)
		}

		ioutil.WriteFile(utils.StatePath(), d, os.ModePerm)
	},
}
