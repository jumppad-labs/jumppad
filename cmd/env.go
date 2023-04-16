package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/shipyard-run/hclconfig/types"
	"github.com/shipyard-run/shipyard/pkg/config/resources"
	"github.com/shipyard-run/shipyard/pkg/shipyard"
	"github.com/spf13/cobra"
)

func newEnvCmd(e shipyard.Engine) *cobra.Command {
	var unset bool

	envCmd := &cobra.Command{
		Use:   "env",
		Short: "Prints environment variables defined by the blueprint",
		Long:  "Prints environment variables defined by the blueprint",
		Example: `
  # Display environment variables
  shipyard env
  
  VAR1=value
  VAR2=value
  
  # Set environment variables on Linux based systems
  eval $(shipyard env)
    
  # Set environment variables on Windows based systems
  Invoke-Expression "shipyard env" | ForEach-Object { Invoke-Expression $_ }

  # Unset environment variables on Linux based systems
  eval $(shipyard env --unset)

  # Unset environment variables on Windows based systems
  Invoke-Expression "shipyard env --unset" | ForEach-Object { Remove-Item $_ }
`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {

			// load the stack
			c, err := resources.LoadState()
			if err != nil {
				cmd.Println("Error: Unable to load state, ", err)
				os.Exit(1)
			}

			prefix := "export "
			if unset {
				prefix = "unset "
			}
			if runtime.GOOS == "windows" {
				prefix = "$Env:"
				if unset {
					prefix = "Env:\\"
				}
			}

			// add output variables
			for _, r := range c.Resources {
				if r.Metadata().Type == types.TypeOutput {
					if r.Metadata().Disabled {
						continue
					}

					if r.Metadata().Module != "" {
						continue
					}

					val := strings.ReplaceAll(r.(*types.Output).Value, `\`, `\\`)
					if unset {
						fmt.Printf("%s%s\n", prefix, r.Metadata().Name)
					} else {
						fmt.Printf("%s%s=\"%s\"\n", prefix, r.Metadata().Name, val)
					}
				}
			}
			return nil
		},
		SilenceUsage: true,
	}

	envCmd.Flags().BoolVarP(&unset, "unset", "", false, "When set to true Shipyard will print unset commands for environment variables defined by the blueprint")
	return envCmd
}
