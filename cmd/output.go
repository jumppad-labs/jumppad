package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/hokaccha/go-prettyjson"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/spf13/cobra"
)

var envFlag bool

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
			if r.Metadata().Type == types.TypeOutput {
				// don't output when disabled
				if r.GetDisabled() {
					continue
				}

				if r.Metadata().Module != "" {
					continue
				}

				if jsonFlag {
					out[r.Metadata().Name] = r.(*types.Output).Value
				} else if envFlag {
					fmt.Println(GrayText.Render("export ") + WhiteText.Render(r.Metadata().Name) + GrayText.Render("=") + GreenIcon.Render(fmt.Sprintf("\"%s\"", (r.(*types.Output).Value.(string)))))
				} else {
					fmt.Println(WhiteText.Render(r.Metadata().Name) + GrayText.Render("=") + GreenIcon.Render(r.(*types.Output).Value.(string)))
				}
			}
		}

		if jsonFlag {
			formatter := prettyjson.Formatter{
				Indent:          2,
				KeyColor:        color.New(color.FgWhite, color.Bold),
				StringColor:     color.New(color.FgGreen, color.Bold),
				BoolColor:       color.New(color.FgGreen, color.Bold),
				NumberColor:     color.New(color.FgGreen, color.Bold),
				NullColor:       color.New(color.FgBlack, color.Bold),
				DisabledColor:   false,
				StringMaxLength: 0,
				Newline:         "\n",
			}

			d, _ := formatter.Marshal(out)
			fmt.Printf("%s\n", string(d))
		}
	},
}

func init() {
	outputCmd.Flags().BoolVarP(&jsonFlag, "json", "", false, "Output the output as JSON")
	outputCmd.Flags().BoolVarP(&envFlag, "env", "", false, "Output the output as environment variables")
}
