package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/hokaccha/go-prettyjson"
	"github.com/shipyard-run/hclconfig"
	"github.com/shipyard-run/hclconfig/types"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"
)

var outputCmd = &cobra.Command{
	Use:   "output",
	Short: "Show the output variables",
	Long:  `Show the output variables`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// load the stack
		p := hclconfig.NewParser(hclconfig.DefaultOptions())
		d, err := ioutil.ReadFile(utils.StatePath())
		if err != nil {
			fmt.Println("Unable to read state file")
			os.Exit(1)
		}

		cfg, err := p.UnmarshalJSON(d)
		if err != nil {
			fmt.Println("Unable to unmarshal state file")
			os.Exit(1)
		}

		out := map[string]string{}
		// get the output variables
		for _, r := range cfg.Resources {
			if r.Metadata().Type == types.TypeOutput {
				// don't output when disabled
				if r.Metadata().Disabled {
					continue
				}

				out[r.Metadata().Name] = r.(*types.Output).Value

				if len(args) > 0 && strings.ToLower(args[0]) == strings.ToLower(r.Metadata().Name) {
					cmd.Println(r.(*types.Output).Value)
					return
				}
			}
		}

		s, _ := prettyjson.Marshal(out)
		cmd.Println(string(s))
	},
}
