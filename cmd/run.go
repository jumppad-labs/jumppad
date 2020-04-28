package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
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

func newRunCmd(e shipyard.Engine, bp clients.Getter, hc clients.HTTP, bc clients.System, l hclog.Logger) *cobra.Command {
	var noOpen bool
	var force bool
	runCmd := &cobra.Command{
		Use:   "run [file] [directory] ...",
		Short: "Run the supplied stack configuration",
		Long:  `Run the supplied stack configuration`,
		Example: `
  # Recursively create a stack from a directory
  shipyard run ./-stack

  # Create a stack from a specific file
  shipyard run my-stack/network.hcl
  
  # Create a stack from a blueprint in GitHub
  shipyard run github.com/shipyard-run/blueprints//vault-k8s
	`,
		Args:         cobra.ArbitraryArgs,
		RunE:         newRunCmdFunc(e, bp, hc, bc, &noOpen, &force, l),
		SilenceUsage: true,
	}
	runCmd.Flags().BoolVarP(&noOpen, "no-browser", "", false, "When set to true Shipyard does not open the browser windows defined in the blueprint")
	runCmd.Flags().BoolVarP(&force, "force-update", "", false, "When set to true Shipyard will ignore cached images or files and will download all resources")

	return runCmd
}

func newRunCmdFunc(e shipyard.Engine, bp clients.Getter, hc clients.HTTP, bc clients.System, noOpen *bool, force *bool, l hclog.Logger) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if *force == true {
			bp.SetForce(true)
			e.GetClients().ContainerTasks.SetForcePull(true)
		}

		// Check the system to see if Docker is running and everything is installed
		s, err := bc.Preflight()
		if err != nil {
			fmt.Println("")
			fmt.Println("###### SYSTEM DIAGNOSTICS ######")
			fmt.Println(s)
			return err
		}

		// check the shipyard version
		text, ok := bc.CheckVersion(version)
		if !ok {
			fmt.Println("")
			fmt.Println(text)
			fmt.Println("")
		}

		// create the shipyard home
		os.MkdirAll(utils.ShipyardHome(), os.FileMode(0755))

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

		// do not open the browser windows
		if *noOpen == false {

			browserList := []string{}

			// check if blueprint is in the state, if so do not open these windows again
			if !blueprintExists && e.Blueprint() != nil {
				browserList = e.Blueprint().BrowserWindows
			}

			// check for browser windows in the applied resources
			for _, r := range res {
				switch r.Info().Type {
				case config.TypeContainer:
					c := r.(*config.Container)
					for _, p := range c.Ports {
						if p.Host != "" && p.OpenInBrowser != "" {
							browserList = append(browserList, buildBrowserPath(r.Info().Name, p.Host, r.Info().Type, p.OpenInBrowser))
						}
					}
				case config.TypeIngress:
					c := r.(*config.Ingress)
					for _, p := range c.Ports {
						if p.Host != "" && p.OpenInBrowser != "" {
							browserList = append(browserList, buildBrowserPath(r.Info().Name, p.Host, r.Info().Type, p.OpenInBrowser))
						}
					}
				case config.TypeContainerIngress:
					c := r.(*config.ContainerIngress)
					for _, p := range c.Ports {
						if p.Host != "" && p.OpenInBrowser != "" {
							browserList = append(browserList, buildBrowserPath(r.Info().Name, p.Host, r.Info().Type, p.OpenInBrowser))
						}
					}
				case config.TypeNomadIngress:
					c := r.(*config.NomadIngress)
					for _, p := range c.Ports {
						if p.Host != "" && p.OpenInBrowser != "" {
							browserList = append(browserList, buildBrowserPath(r.Info().Name, p.Host, r.Info().Type, p.OpenInBrowser))
						}
					}
				case config.TypeK8sIngress:
					c := r.(*config.K8sIngress)
					for _, p := range c.Ports {
						if p.Host != "" && p.OpenInBrowser != "" {
							browserList = append(browserList, buildBrowserPath(r.Info().Name, p.Host, r.Info().Type, p.OpenInBrowser))
						}
					}
				case config.TypeDocs:
					c := r.(*config.Docs)
					if c.OpenInBrowser {
						browserList = append(browserList, buildBrowserPath(r.Info().Name, strconv.Itoa(c.Port), r.Info().Type, ""))
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
						be := bc.OpenBrowser(uri)
						if be != nil {
							l.Error("Unable to open browser", "error", be)
						}
					}

					wg.Done()
				}(b)
			}

			wg.Wait()

		}

		// if we have a blueprint show the header
		if e.Blueprint() != nil {
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

			if len(e.Blueprint().Environment) > 0 {
				cmd.Println("This blueprint defines the following environment varaibles:")
				cmd.Println("")
				for _, env := range e.Blueprint().Environment {
					cmd.Printf("%s=%s\n", env.Key, env.Value)
				}
				cmd.Println("")
				cmd.Println("You can set these using the following command:")

				if runtime.GOOS == "windows" {
					cmd.Println(`@FOR /f "tokens=*" %i IN ('minikube -p minikube docker-env') DO @%i`)
				} else {
					cmd.Println("eval $(shipyard env)")
				}
			}
		}

		return nil
	}
}

func buildBrowserPath(n, p string, t config.ResourceType, path string) string {
	ty := t
	if t == config.TypeNomadIngress || t == config.TypeContainerIngress || t == config.TypeK8sIngress {
		ty = config.TypeIngress
	}

	return fmt.Sprintf("http://%s.%s.shipyard.run:%s%s", n, ty, p, path)
}

func bluePrintInState() bool {
	//load the state
	sc := config.New()
	sc.FromJSON(utils.StatePath())

	return sc.Blueprint != nil
}
