package cmd

import (
	"errors"
	"fmt"
	"os"

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
		Run: func(cmd *cobra.Command, args []string) {
			var err error
			dst := args[0]
			fmt.Println("Fetching blueprint from: ", dst)
			fmt.Println("")

			// create a logger
			log := createLogger()

			// create the shipyard home
			os.MkdirAll(utils.ShipyardHome(), os.FileMode(0755))

			if utils.IsLocalFolder(dst) {
				log.Error("Parameter is not a remote blueprint, e.g. github.com/shipyard-run/blueprints//vault-k8s")
				os.Exit(1)
			}

			// fetch the remote server from github
			err = bp.Get(dst, utils.GetBlueprintLocalFolder(dst))
			if err != nil {
				log.Error("Unable to retrieve blueprint", "error", err)
				os.Exit(1)
			}
		},
	}
}
