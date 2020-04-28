package cmd

import (
	"fmt"
	"os"

	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/shipyard"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"
)

func newEnvCmd(e shipyard.Engine) *cobra.Command {
	envCmd := &cobra.Command{
		Use:   "env",
		Short: "Prints environment variables defined by the blueprint",
		Long:  "Prints environment variables defined by the blueprint",
		Example: `
  # Display environment variables
	shipyard env

	VAR1=value
	VAR2=value

	# Set environemnt variables
	eval $(shipyard env)
	`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {

			c := config.New()
			err := c.FromJSON(utils.StatePath())
			if err != nil {
				fmt.Println("Unable to load state", err)
				os.Exit(1)
			}

			if c.Blueprint != nil && len(c.Blueprint.Environment) > 0 {
				for _, env := range c.Blueprint.Environment {
					fmt.Printf("%s=%s\n", env.Key, env.Value)
				}
			}
			return nil
		},
		SilenceUsage: true,
	}

	return envCmd
}
