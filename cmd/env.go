package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/jumppad"
	"github.com/spf13/cobra"
)

func newEnvCmd(e jumppad.Engine) *cobra.Command {
	var unset bool

	envCmd := &cobra.Command{
		Use:   "env",
		Short: "Prints environment variables defined by the blueprint",
		Long:  "Prints environment variables defined by the blueprint",
		Example: `
  # Display environment variables
  jumppad env
  
  VAR1=value
  VAR2=value
  
  # Set environment variables on Linux based systems
  eval $(jumppad env)
    
  # Set environment variables on Windows based systems
  Invoke-Expression "jumppad env" | ForEach-Object { Invoke-Expression $_ }

  # Unset environment variables on Linux based systems
  eval $(jumppad env --unset)

  # Unset environment variables on Windows based systems
  Invoke-Expression "jumppad env --unset" | ForEach-Object { Remove-Item $_ }
`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {

			// load the stack
			c, err := config.LoadState()
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
				if r.Metadata().ResourceType == types.TypeOutput {
					if r.Metadata().Disabled {
						continue
					}

					if r.Metadata().ResourceModule != "" {
						continue
					}

					d, _ := json.Marshal(r.(*types.Output).Value)

					// trim any strings that wrap the output from marshaling
					val := strings.TrimPrefix(string(d), "\"")
					val = strings.TrimSuffix(val, "\"")

					// escape any json
					val = strings.ReplaceAll(val, `\`, `\\`)
					val = strings.ReplaceAll(val, `"`, `\"`)
					if unset {
						fmt.Printf("%s%s\n", prefix, r.Metadata().ResourceName)
					} else {
						fmt.Printf("%s%s=\"%s\"\n", prefix, r.Metadata().ResourceName, val)
					}
				}
			}
			return nil
		},
		SilenceUsage: true,
	}

	envCmd.Flags().BoolVarP(&unset, "unset", "", false, "When set to true jumppad will print unset commands for environment variables defined by the blueprint")
	return envCmd
}
