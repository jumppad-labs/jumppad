package cmd

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/xerrors"

	"context"

	getter "github.com/hashicorp/go-getter"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"
)

// ErrorInvalidBlueprintURI is returned when the URI for a blueprint can not be parsed
var ErrorInvalidBlueprintURI = errors.New("error invalid Blueprint URI, blueprints should be formatted 'github.com/org/repo//blueprint'")

var getCmd = &cobra.Command{
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
		dst, err = pullRemoteBlueprint(dst)
		if err != nil {
			log.Error("Unable to retrieve blueprint", "error", err)
			os.Exit(1)
		}
	},
}

// pullRemoteBlueprint attempts to retrieve a blueprint
// from a remote location
// returns the local path if successful or error on
// failure
func pullRemoteBlueprint(uri string) (string, error) {

	bpFolder, err := utils.GetBlueprintFolder(uri)
	if err != nil {
		return "", err
	}

	dst := fmt.Sprintf("%s/.shipyard/blueprints/%s", os.Getenv("HOME"), bpFolder)

	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// if the argument is a url fetch it first
	c := &getter.Client{
		Ctx:     context.Background(),
		Src:     uri,
		Dst:     dst,
		Pwd:     pwd,
		Mode:    getter.ClientModeAny,
		Options: []getter.ClientOption{},
	}

	err = c.Get()
	if err != nil {
		return "", xerrors.Errorf("unable to fetch blueprint from %s: %w", uri, err)
	}

	return dst, nil
}
