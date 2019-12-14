package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/otiai10/copy"

	"golang.org/x/xerrors"

	"context"

	getter "github.com/hashicorp/go-getter"
	"github.com/shipyard-run/cli/pkg/shipyard"
	"github.com/spf13/cobra"
)

// ErrorInvalidBlueprintURI is returned when the URI for a blueprint can not be parsed
var ErrorInvalidBlueprintURI = errors.New("error invalid Blueprint URI, blueprints should be formatted 'github.com/org/repo//blueprint'")

var applyCmd = &cobra.Command{
	Use:   "apply [file] [directory] ...",
	Short: "Apply the supplied stack configuration",
	Long:  `Apply the supplied stack configuration`,
	Example: `  # Recursively create a stack from a directory
  yard apply my-stack

  # Create a stack from a specific file
  yard apply my-stack/network.hcl
	`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		dst := args[0]

		if !IsLocalFolder(dst) {
			// fetch the remote server from github
			dst, err = pullRemoteBlueprint(dst)
			if err != nil {
				panic(err)
			}
		}

		// Load the files
		e, err := shipyard.NewWithFolder(dst)
		if err != nil {
			panic(err)
		}

		// if we have a blueprint show the header
		if e.Blueprint() != nil {
			fmt.Println("Title", e.Blueprint().Title)
			fmt.Println("Author", e.Blueprint().Author)
			fmt.Println("")
			fmt.Println(e.Blueprint().Intro)
			fmt.Println("")
		}

		err = e.Apply()
		if err != nil {
			panic(err)
		}

		// copy the blueprints to our state folder
		// this is temporary
		err = copy.Copy(dst, StateDir())
		if err != nil {
			panic(err)
		}

		// open any browser windows
		if e.Blueprint() != nil {
			openCommand := "open"
			if runtime.GOOS == "linux" {
				openCommand = "xdg-open"
			}

			for _, b := range e.Blueprint().BrowserWindows {
				cmd := exec.Command(openCommand, b)
				cmd.Run()
			}
		}
	},
}

// pullRemoteBlueprint attempts to retrieve a blueprint
// from a remote location
// returns the local path if successful or error on
// failure
func pullRemoteBlueprint(uri string) (string, error) {

	bpFolder, err := GetBlueprintFolder(uri)
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
