package cmd

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"

	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
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

		// have we already got a blueprint in the state
		blueprintExists := false
		if bluePrintInState() {
			blueprintExists = true
		}

		// Load the files
		res, err := e.Apply(dst)
		if err != nil {
			return fmt.Errorf("Unable to apply blueprint: %s", err)
		}

		// if we have a blueprint show the header
		if e.Blueprint() != nil {

			// do not open the browser windows
			if *noOpen == false {

				browserList := e.Blueprint().BrowserWindows

				// check if blueprint is in the state, if so do not open these windows again
				if blueprintExists {
					browserList = []string{}
				}

				// check for browser windows in the applied resources
				for _, r := range res {
					switch r.Info().Type {
					case config.TypeContainer:
						c := r.(*config.Container)
						for _, p := range c.Ports {
							if p.Host != "" && p.OpenInBrowser {
								browserList = append(browserList, fmt.Sprintf("http://localhost:%s", p.Host))
							}
						}
					case config.TypeIngress:
						c := r.(*config.Ingress)
						for _, p := range c.Ports {
							if p.Host != "" && p.OpenInBrowser {
								browserList = append(browserList, fmt.Sprintf("http://localhost:%s", p.Host))
							}
						}
					case config.TypeContainerIngress:
						c := r.(*config.ContainerIngress)
						for _, p := range c.Ports {
							if p.Host != "" && p.OpenInBrowser {
								browserList = append(browserList, fmt.Sprintf("http://localhost:%s", p.Host))
							}
						}
					case config.TypeNomadIngress:
						c := r.(*config.NomadIngress)
						for _, p := range c.Ports {
							if p.Host != "" && p.OpenInBrowser {
								browserList = append(browserList, fmt.Sprintf("http://localhost:%s", p.Host))
							}
						}
					case config.TypeK8sIngress:
						c := r.(*config.K8sIngress)
						for _, p := range c.Ports {
							if p.Host != "" && p.OpenInBrowser {
								browserList = append(browserList, fmt.Sprintf("http://localhost:%s", p.Host))
							}
						}
					case config.TypeDocs:
						c := r.(*config.Docs)
						if c.OpenInBrowser {
							browserList = append(browserList, fmt.Sprintf("http://localhost:%d", c.Port))
						}
					}
				}

				// check the browser windows in the blueprint file
				wg := sync.WaitGroup{}

				for _, b := range browserList {
					wg.Add(1)
					go func(uri string) {
						// health check the URL
						err := hc.HealthCheckHTTP(uri, 30*time.Second)
						if err == nil {
							be := bc.Open(uri)
							if be != nil {
								l.Error("Unable to open browser", "error", be)
							}
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

func bluePrintInState() bool {
	//load the state
	sc := config.New()
	sc.FromJSON(utils.StatePath())

	return sc.Blueprint != nil
}
