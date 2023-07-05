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
	"github.com/jumppad-labs/hclconfig/types"
	gvm "github.com/shipyard-run/version-manager"

	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/jumppad-labs/jumppad/pkg/jumppad"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/spf13/cobra"

	markdown "github.com/MichaelMure/go-term-markdown"
)

func newRunCmd(e jumppad.Engine, bp clients.Getter, hc clients.HTTP, bc clients.System, vm gvm.Versions, cc clients.Connector, l hclog.Logger) *cobra.Command {
	var noOpen bool
	var force bool
	var y bool
	var runVersion string
	var variables []string
	var variablesFile string

	runCmd := &cobra.Command{
		Use:   "up [file] | [directory]",
		Short: "Create the resources at the given path",
		Long:  `Create the resources at the given path`,
		Example: `
  # Create resources from .hcl files in the current folder
  jumppad up ./

  # Create resources from a specific file
  jumppad up my-stack/network.hcl

  # Create resources from a blueprint in GitHub
  jumppad up github.com/jumppad-labs/blueprints/kubernetes-vault
	`,
		Args:         cobra.ArbitraryArgs,
		RunE:         newRunCmdFunc(e, bp, hc, bc, vm, cc, &noOpen, &force, &runVersion, &y, &variables, &variablesFile, l),
		SilenceUsage: true,
	}

	runCmd.Flags().BoolVarP(&noOpen, "no-browser", "", false, "When set to true Jumppad will not open the browser windows defined in the blueprint")
	runCmd.Flags().BoolVarP(&force, "force-update", "", false, "When set to true Jumppad ignores cached images or files and will download all resources")
	runCmd.Flags().StringSliceVarP(&variables, "var", "", nil, "Allows setting variables from the command line, variables are specified as a key and value, e.g --var key=value. Can be specified multiple times")
	runCmd.Flags().StringVarP(&variablesFile, "vars-file", "", "", "Load variables from a location other than *.vars files in the blueprint folder. E.g --vars-file=./file.vars")

	return runCmd
}

func newRunCmdFunc(e jumppad.Engine, bp clients.Getter, hc clients.HTTP, bc clients.System, vm gvm.Versions, cc clients.Connector, noOpen *bool, force *bool, runVersion *string, autoApprove *bool, variables *[]string, variablesFile *string, l hclog.Logger) func(cmd *cobra.Command, args []string) error {
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
			return err
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
			return err
		}

		// do not open the browser windows
		if !*noOpen {

			browserList := []string{}
			checkDuration := 30 * time.Second

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
					c := r.(*resources.Ingress)
					if c.OpenInBrowser != "" {
						browserList = append(browserList, buildBrowserPath(r.Metadata().Name, fmt.Sprintf("%d", c.Port), r.Metadata().Type, c.OpenInBrowser))
					}
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
		var blueprint *resources.Blueprint
		bps, _ := e.Config().FindResourcesByType(resources.TypeBlueprint)
		for _, bp := range bps {
			// pick the first blueprint in the root
			if bp.Metadata().Module == "" {
				blueprint = bp.(*resources.Blueprint)
				break
			}
		}

		if blueprint != nil {
			cmd.Println("")
			cmd.Println("########################################################")
			cmd.Println("")
			cmd.Println("Title", blueprint.Title)
			cmd.Println("Author", blueprint.Author)

			// parse the body as markdown and print
			intro := markdown.Render(blueprint.Description, 80, 0)

			cmd.Println("")
			cmd.Print(string(intro))

			outputs := []*types.Output{}
			os, _ := e.Config().FindResourcesByType(types.TypeOutput)
			for _, o := range os {
				// only grab the root outputs
				if o.Metadata().Module == "" {
					outputs = append(outputs, o.(*types.Output))
				}
			}

			if len(outputs) > 0 {
				cmd.Println("")
				cmd.Printf("This blueprint defines %d output variables.\n", len(outputs))
				cmd.Println("")

				maxLen := 0
				for _, o := range outputs {
					if len(o.Name) > maxLen {
						maxLen = len(o.Name)
					}
				}

				format := fmt.Sprintf(" * %%%ds: %%s\n", maxLen)

				for _, o := range outputs {
					fmt.Printf(format, o.Name, o.Value)
				}

				cmd.Println("")
				cmd.Println("You can set output variables as environment variables for your current terminal session using the following command:")
				cmd.Println("")

				if runtime.GOOS == "windows" {
					cmd.Println(`Invoke-Expression "jumppad env" | ForEach-Object { Invoke-Expression $_ }`)
				} else {
					cmd.Println("eval $(jumppad env)")
				}
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

	return fmt.Sprintf("http://%s:%s%s", utils.FQDN(n, "", ty), p, path)
}
