package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/hokaccha/go-prettyjson"
	"github.com/shipyard-run/shipyard/pkg/config"
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
		c := config.New()
		err := c.FromJSON(utils.StatePath())
		if err != nil {
			fmt.Println("Unable to load state", err)
			os.Exit(1)
		}

		out := map[string]string{}
		// get the output variables
		for _, r := range c.Resources {
			if r.Info().Type == config.TypeOutput {
				out[r.Info().Name] = r.(*config.Output).Value

				if len(args) > 0 && strings.ToLower(args[0]) == strings.ToLower(r.Info().Name) {
					cmd.Println(r.(*config.Output).Value)
					return
				}
			}
		}

		s, _ := prettyjson.Marshal(out)
		cmd.Println(string(s))
	},
}
