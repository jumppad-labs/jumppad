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
	"github.com/shipyard-run/shipyard/pkg/shipyard"
	"github.com/spf13/cobra"
)

// ErrorInvalidBlueprintURI is returned when the URI for a blueprint can not be parsed
var ErrorInvalidBlueprintURI = errors.New("error invalid Blueprint URI, blueprints should be formatted 'github.com/org/repo//blueprint'")

var runCmd = &cobra.Command{
	Use:   "run [file] [directory] ...",
	Short: "Run the supplied stack configuration",
	Long:  `Run the supplied stack configuration`,
	Example: `
  # Recursively create a stack from a directory
  yard run ./-stack

  # Create a stack from a specific file
  yard run my-stack/network.hcl
  
  # Create a stack from a blueprint in GitHub
  yard run github.com/shipyard-run/blueprints//vault-k8s
	`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		dst := args[0]

		fmt.Println("Running configuration from: ", dst)
		fmt.Println("")

		// create a logger
		log := createLogger()

		if !IsLocalFolder(dst) {
			// fetch the remote server from github
			dst, err = pullRemoteBlueprint(dst)
			if err != nil {
				log.Error("Unable to retrieve blueprint", "error", err)
				return
			}
		}

		// Load the files
		e, err := shipyard.NewWithFolder(dst, log)
		if err != nil {
			log.Error("Unable to load blueprint", "error", err)
			return
		}

		// if we have a blueprint show the header
		if e.Blueprint() != nil {
			fmt.Println("Title", e.Blueprint().Title)
			fmt.Println("Author", e.Blueprint().Author)
			fmt.Println("")
		}

		fmt.Printf("Creating %d resources\n\n", e.ResourceCount())

		err = e.Apply()
		if err != nil {
			log.Error("Unable to apply blueprint", "error", err)

			log.Info("Attempting to roll back state")
			err := e.Destroy()
			if err != nil {
				log.Error("Unable to roll back state, you may need to manually remove Docker containers and networks", "error", err)
			}

			return
		}

		// copy the blueprints to our state folder
		// this is temporary
		err = copy.Copy(dst, StateDir())
		if err != nil {
			log.Error("Unable to copy blueprint to state folder", "error", err)
			return
		}

		if e.Blueprint() != nil {
			fmt.Println("")
			fmt.Println(e.Blueprint().Intro)
			fmt.Println("")
		}

		// apply any env vars
		/*
			if e.Blueprint() != nil && len(e.Blueprint().Environment) > 0 {
				fmt.Println("")
				fmt.Println("Setting environment variables:")
				fmt.Println("")
				ef, err := NewEnv(fmt.Sprintf("%s/env.var", StateDir()))
				if err != nil {
					panic(err)
				}
				defer ef.Close()

				for _, e := range e.Blueprint().Environment {
					fmt.Printf("export %s=%s\n", e.Key, e.Value)
					err := ef.Set(e.Key, e.Value)
					if err != nil {
						panic(err)
					}
				}

				fmt.Println("")
				fmt.Println("environment variables will be restored to previous values when using the `yard delete` command")
			}
		*/

		// open any browser windows
		//TODO implement windows using start "start http://www.google.com"
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
