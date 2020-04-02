package cmd

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"

	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/shipyard"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"

	markdown "github.com/MichaelMure/go-term-markdown"
)

func newRunCmd(e shipyard.Engine, bp clients.Blueprints, hc clients.HTTP, bc clients.Browser, l hclog.Logger) *cobra.Command {
	var noOpen bool
	runCmd := &cobra.Command{
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
		RunE: newRunCmdFunc(e, bp, hc, bc, &noOpen, l),
	}
	runCmd.Flags().BoolVarP(&noOpen, "no-browser", "", false, "When set to true Shipyard does not open the browser windows defined in the blueprint")

	return runCmd
}

func newRunCmdFunc(e shipyard.Engine, bp clients.Blueprints, hc clients.HTTP, bc clients.Browser, noOpen *bool, l hclog.Logger) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// create the shipyard home
		os.MkdirAll(utils.ShipyardHome(), os.FileMode(0755))

		var err error
		dst := ""
		if len(args) == 1 {
			dst = args[0]
		} else {
			dst = "./"
		}

		if dst == "." {
			dst = "./"
		}

		if dst != "" {
			cmd.Println("Running configuration from: ", dst)
			cmd.Println("")

			if !utils.IsLocalFolder(dst) && !utils.IsHCLFile(dst) {
				// fetch the remote server from github
				err := bp.Get(dst, utils.GetBlueprintLocalFolder(dst))
				if err != nil {
					return fmt.Errorf("Unable to retrieve blueprint: %s", err)
				}

				dst = utils.GetBlueprintLocalFolder(dst)
			}
		}

		// Load the files
		err = e.Apply(dst)
		if err != nil {
			return fmt.Errorf("Unable to apply blueprint: %s", err)
		}

		// if we have a blueprint show the header
		if e.Blueprint() != nil {

			// do not open the browser windows
			if *noOpen == false {

				wg := sync.WaitGroup{}

				for _, b := range e.Blueprint().BrowserWindows {
					wg.Add(1)
					go func(uri string) {
						// health check the URL
						err := hc.HealthCheckHTTP(uri, 30*time.Second)
						if err == nil {
							bc.Open(uri)
						}

						wg.Done()
					}(b)
				}

				wg.Wait()
			}

			cmd.Println("")
			cmd.Println("########################################################")
			cmd.Println("")
			cmd.Println("Title", e.Blueprint().Title)
			cmd.Println("Author", e.Blueprint().Author)
			cmd.Println("")
			cmd.Println("########################################################")

			// parse the body as markdown and print
			intro := markdown.Render(e.Blueprint().Intro, 80, 0)

			cmd.Println("")
			cmd.Println("")
			cmd.Print(string(intro))
			cmd.Println("")
		}

		return nil
	}
}
