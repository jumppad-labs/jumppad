package cmd

import (
	"os"
	"strings"

	"github.com/hokaccha/go-prettyjson"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/spf13/cobra"
)

var outputCmd = &cobra.Command{
	Use:   "output",
	Short: "Show the output variables",
	Long:  `Show the output variables`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// load the stack
		cfg, err := resources.LoadState()
		if err != nil {
			cmd.Println("Error: Unable to load state, ", err)
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

				if r.Metadata().Module != "" {
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
