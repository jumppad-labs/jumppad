package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/hokaccha/go-prettyjson"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/spf13/cobra"
)

var outputCmd = &cobra.Command{
	Use:   "output",
	Short: "Show the output variables",
	Long:  `Show the output variables`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// load the stack
		cfg, err := config.LoadState()
		if err != nil {
			cmd.Println("Error: Unable to load state, ", err)
			os.Exit(1)
		}

		out := map[string]interface{}{}
		// get the output variables
		for _, r := range cfg.Resources {
			if r.Metadata().ResourceType == types.TypeOutput {
				// don't output when disabled
				if r.Metadata().Disabled {
					continue
				}

				if r.Metadata().ResourceModule != "" {
					continue
				}

				out[r.Metadata().ResourceName] = r.(*types.Output).Value

				if len(args) > 0 && strings.EqualFold(args[0], r.Metadata().ResourceName) {
					d, _ := json.Marshal(r.(*types.Output).Value)
					fmt.Printf("%s", string(d))
					return
				}
			}
		}

		d, _ := prettyjson.Marshal(out)
		fmt.Printf("%s", string(d))
	},
}
