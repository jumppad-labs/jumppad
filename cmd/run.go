package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"

	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/shipyard"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"
)

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
		dst := ""
		if len(args) == 1 {
			dst = args[0]
		}

		// create a logger
		log := createLogger()

		if dst != "" {
			fmt.Println("Running configuration from: ", dst)
			fmt.Println("")

			// create the shipyard home
			os.MkdirAll(utils.ShipyardHome(), os.FileMode(0755))

			if !utils.IsLocalFolder(dst) && !utils.IsHCLFile(dst) {
				// fetch the remote server from github
				dst, err = pullRemoteBlueprint(dst)
				if err != nil {
					log.Error("Unable to retrieve blueprint", "error", err)
					return
				}
			}
		}

		// Load the files
		var e *shipyard.Engine
		e, err = shipyard.New(log)
		if err != nil {
			log.Error("Unable to load config", "error", err)
			return
		}

		err = e.Apply(dst)
		if err != nil {
			log.Error("Unable to apply blueprint", "error", err)
			return
		}

		// if we have a blueprint show the header
		if e.Blueprint() != nil {
			fmt.Println("")
			fmt.Println("########################################################")
			fmt.Println("")
			fmt.Println("Title", e.Blueprint().Title)
			fmt.Println("Author", e.Blueprint().Author)
			fmt.Println("")

			fmt.Println("")
			fmt.Println(e.Blueprint().Intro)
			fmt.Println("")

			openCommand := "open"
			if runtime.GOOS == "linux" {
				openCommand = "xdg-open"
			}

			c := clients.NewHTTP(1*time.Second, hclog.Default())
			wg := sync.WaitGroup{}

			for _, b := range e.Blueprint().BrowserWindows {
				wg.Add(1)
				go func(uri string) {
					// health check the URL
					err := c.HealthCheckHTTP(uri, 30*time.Second)
					if err == nil {
						cmd := exec.Command(openCommand, uri)
						cmd.Run()
					}

					wg.Done()
				}(b)
			}

			wg.Wait()
		}
	},
}
