package cmd

import (
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	gvm "github.com/shipyard-run/version-manager"

	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/jumppad-labs/jumppad/pkg/shipyard"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/spf13/cobra"

	markdown "github.com/MichaelMure/go-term-markdown"
)

func newRunCmd(e shipyard.Engine, bp clients.Getter, hc clients.HTTP, bc clients.System, vm gvm.Versions, cc clients.Connector, l hclog.Logger) *cobra.Command {
	var noOpen bool
	var force bool
	var y bool
	var runVersion string
	var variables []string
	var variablesFile string

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
		RunE:         newRunCmdFunc(e, bp, hc, bc, vm, cc, &noOpen, &force, &runVersion, &y, &variables, &variablesFile, l),
		SilenceUsage: true,
	}

	runCmd.Flags().StringVarP(&runVersion, "version", "v", "", "When set, run creates the specified resources using a particular Shipyard version")
	runCmd.Flags().BoolVarP(&y, "y", "y", false, "When set, Shipyard will not prompt for confirmation")
	runCmd.Flags().BoolVarP(&noOpen, "no-browser", "", false, "When set to true Shipyard will not open the browser windows defined in the blueprint")
	runCmd.Flags().BoolVarP(&force, "force-update", "", false, "When set to true Shipyard ignores cached images or files and will download all resources")
	runCmd.Flags().StringSliceVarP(&variables, "var", "", nil, "Allows setting variables from the command line, variables are specified as a key and value, e.g --var key=value. Can be specified multiple times")
	runCmd.Flags().StringVarP(&variablesFile, "vars-file", "", "", "Load variables from a location other than *.vars files in the blueprint folder. E.g --vars-file=./file.vars")

	return runCmd
}

func newRunCmdFunc(e shipyard.Engine, bp clients.Getter, hc clients.HTTP, bc clients.System, vm gvm.Versions, cc clients.Connector, noOpen *bool, force *bool, runVersion *string, autoApprove *bool, variables *[]string, variablesFile *string, l hclog.Logger) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// create the shipyard and sub folders in the users home directory
		utils.CreateFolders()

		if *force == true {
			bp.SetForce(true)
			e.GetClients().ContainerTasks.SetForcePull(true)
		}

		// parse the vars into a map
		vars := map[string]string{}
		for _, v := range *variables {
			parts := strings.Split(v, "=")
			if len(parts) == 2 {
				vars[parts[0]] = parts[1]
			}
		}

		// Check the system to see if Docker is running and everything is installed
		s, err := bc.Preflight()
		if err != nil {
			cmd.Println("")
			cmd.Println("###### SYSTEM DIAGNOSTICS ######")
			cmd.Println(s)
			return err
		}

		// check the variables file exists
		if variablesFile != nil && *variablesFile != "" {
			if _, err := os.Stat(*variablesFile); err != nil {
				return fmt.Errorf("Variables file %s, does not exist", *variablesFile)
			}
		} else {
			vf := ""
			variablesFile = &vf
		}

		// create the certificates for the connector
		if cb, err := cc.GetLocalCertBundle(utils.CertsDir("")); err != nil || cb == nil {
			// generate certs
			l.Debug("Generating TLS Certificates for Ingress", "path", utils.CertsDir(""))
			_, err := cc.GenerateLocalCertBundle(utils.CertsDir(""))
			if err != nil {
				return fmt.Errorf("Unable to generate connector certificates: %s", err)
			}
		}

		// start the connector
		if !cc.IsRunning() {
			cb, err := cc.GetLocalCertBundle(utils.CertsDir(""))
			if err != nil {
				return fmt.Errorf("Unable to get certificates to secure ingress: %s", err)
			}

			l.Debug("Starting API server")

			err = cc.Start(cb)
			if err != nil {
				return fmt.Errorf("Unable to start API server: %s", err)
			}
		}

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

		// Parse the config to check it is valid
		_, err = e.ParseConfigWithVariables(dst, vars, *variablesFile)
		if err != nil {
			return fmt.Errorf("Unable to read config: %s", err)
		}

		// have we already got a blueprint in the state
		blueprintExists := false
		if bluePrintInState() {
			blueprintExists = true
		}

		// update status every 30s to let people know we are still running
		statusUpdate := time.NewTicker(15 * time.Second)
		startTime := time.Now()

		go func() {
			for range statusUpdate.C {
				elapsedTime := time.Since(startTime).Seconds()
				logger.Info(fmt.Sprintf("Please wait, still creating resources [Elapsed Time: %f]", elapsedTime))
			}
		}()

		res, err := e.ApplyWithVariables(dst, vars, *variablesFile)
		if err != nil {
			return fmt.Errorf("Unable to apply blueprint: %s", err)
		}

		// do not open the browser windows
		if *noOpen == false {

			browserList := []string{}
			checkDuration := 30 * time.Second

			// check if blueprint is in the state, if so do not open these windows again
			if !blueprintExists && e.Blueprint() != nil {
				browserList = e.Blueprint().BrowserWindows
				// check for browser windows in the applied resources
				if e.Blueprint().HealthCheckTimeout != "" {
					cd, err := time.ParseDuration(e.Blueprint().HealthCheckTimeout)
					if err == nil {
						checkDuration = cd
					}
				}
			}

			for _, r := range res {
				switch r.Metadata().Type {
				case resources.TypeContainer:
					c := r.(*resources.Container)
					for _, p := range c.Ports {
						if p.Host != "" && p.OpenInBrowser != "" {
							browserList = append(browserList, buildBrowserPath(r.Metadata().Name, p.Host, r.Metadata().Type, p.OpenInBrowser))
						}
					}
				case resources.TypeIngress:
					//c := r.(*resources.Ingress)
					//if c.Source.Driver == resources.IngressSourceLocal && c.Source.Config.OpenInBrowser != "" && c.Source.Config.Port != "" {
					//	browserList = append(browserList, buildBrowserPath(r.Metadata().Name, c.Source.Config.Port, r.Metadata().Type, c.Source.Config.OpenInBrowser))
					//}
				case resources.TypeNomadCluster:
					c := r.(*resources.NomadCluster)
					if c.OpenInBrowser {
						// get the API port
						browserList = append(browserList, buildBrowserPath("server."+r.Metadata().Name, fmt.Sprintf("%d", c.APIPort), r.Metadata().Type, "/"))
					}
				case resources.TypeDocs:
					c := r.(*resources.Docs)
					if c.OpenInBrowser {
						port := strconv.Itoa(c.Port)
						if port == "0" {
							port = "80"
						}

						browserList = append(browserList, buildBrowserPath(r.Metadata().Name, port, r.Metadata().Type, ""))
					}
				}
			}

			// check the browser windows in the blueprint file
			wg := sync.WaitGroup{}
			wg.Add(len(browserList))

			l.Debug("Health check urls for browser windows", "count", len(browserList))
			for _, b := range browserList {
				go func(uri string) {
					// health check the URL
					err := hc.HealthCheckHTTP(uri, []int{200}, checkDuration)
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
			l.Debug("Browser windows open")
		}

		// kill the timer
		statusUpdate.Stop()

		// if we have a blueprint show the header
		if e.Blueprint() != nil {
			cmd.Println("")
			cmd.Println("########################################################")
			cmd.Println("")
			cmd.Println("Title", e.Blueprint().Title)
			cmd.Println("Author", e.Blueprint().Author)

			// parse the body as markdown and print
			intro := markdown.Render(e.Blueprint().Intro, 80, 0)

			cmd.Println("")
			cmd.Print(string(intro))

			if len(e.Blueprint().Environment) > 0 || e.ResourceCountForType("output") > 0 {
				cmd.Println("")
				cmd.Printf("This blueprint defines %d output variables.\n", e.ResourceCountForType("output"))
				cmd.Println("")
				cmd.Println("You can set output variables as environment variables for your current terminal session using the following command:")
				cmd.Println("")

				if runtime.GOOS == "windows" {
					cmd.Println(`Invoke-Expression "shipyard env" | ForEach-Object { Invoke-Expression $_ }`)
				} else {
					cmd.Println("eval $(shipyard env)")
				}
				cmd.Println("")
				cmd.Println("To list output variables use the command:")
				cmd.Println("")
				cmd.Println("shipyard output")
			}
		}

		return nil
	}
}

func buildBrowserPath(n, p string, resourceType string, path string) string {
	// if the path starts with http or https then override the default behaviour
	if strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "http://") {
		// validate this is a good URL
		_, err := url.Parse(path)
		if err == nil {
			return path
		}
	}

	ty := resourceType

	return fmt.Sprintf("http://%s:%s.%s", utils.FQDN(n, "", ty), p, path)
}

func bluePrintInState() bool {
	//load the state
	//sc := config.New()
	//sc.FromJSON(utils.StatePath())

	//return sc.Blueprint != nil
	return false
}
