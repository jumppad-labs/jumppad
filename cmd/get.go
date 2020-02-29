package cmd

import (
	"errors"
	"fmt"

	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"
)

// ErrorInvalidBlueprintURI is returned when the URI for a blueprint can not be parsed
var ErrorInvalidBlueprintURI = errors.New("error invalid Blueprint URI, blueprints should be formatted 'github.com/org/repo//blueprint'")

func newGetCmd(bp clients.Blueprints) *cobra.Command {
	return &cobra.Command{
		Use:   "get [remote blueprint]",
		Short: "Download the blueprint to the Shipyard config folder",
		Long:  `Download the blueprint to the Shipyard configuration folder`,
		Example: `
  # Fetch a blueprint from GitHub
  yard get github.com/shipyard-run/blueprints//vault-k8s
	`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// check the number of args
			if len(args) != 1 {
				return fmt.Errorf("Command takes a single argument")
			}

			var err error
			dst := args[0]
			cmd.Println("Fetching blueprint from: ", dst)
			cmd.Println("")

			if utils.IsLocalFolder(dst) {
				return fmt.Errorf("Parameter is not a remote blueprint, e.g. github.com/shipyard-run/blueprints//vault-k8s")
			}

			// fetch the remote server from github
			err = bp.Get(dst, utils.GetBlueprintLocalFolder(dst))
			if err != nil {
				return fmt.Errorf("Unable to retrieve blueprint: %s", err)
			}

			return nil
		},
	}
}
