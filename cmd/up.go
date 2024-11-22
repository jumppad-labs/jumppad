package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jumppad-labs/hclconfig/resources"

	"github.com/jumppad-labs/jumppad/pkg/clients/connector"
	cclients "github.com/jumppad-labs/jumppad/pkg/clients/container"
	"github.com/jumppad-labs/jumppad/pkg/clients/getter"
	"github.com/jumppad-labs/jumppad/pkg/clients/http"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/clients/system"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/blueprint"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/container"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/docs"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/ingress"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/nomad"
	"github.com/jumppad-labs/jumppad/pkg/jumppad"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/spf13/cobra"

	markdown "github.com/MichaelMure/go-term-markdown"
)

func newRunCmd(e jumppad.Engine, dt cclients.ContainerTasks, bp getter.Getter, hc http.HTTP, bc system.System, cc connector.Connector, l logger.Logger) *cobra.Command {
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
		RunE:         newRunCmdFunc(e, dt, bp, hc, bc, cc, &noOpen, &force, &runVersion, &y, &variables, &variablesFile, l),
		SilenceUsage: true,
	}

	runCmd.Flags().BoolVarP(&noOpen, "no-browser", "", false, "When set to true Jumppad will not open the browser windows defined in the blueprint")
	runCmd.Flags().BoolVarP(&force, "force-update", "", false, "When set to true Jumppad ignores cached images or files and will download all resources")
	runCmd.Flags().StringSliceVarP(&variables, "var", "", nil, "Allows setting variables from the command line, variables are specified as a key and value, e.g --var key=value. Can be specified multiple times")
	runCmd.Flags().StringVarP(&variablesFile, "vars-file", "", "", "Load variables from a location other than *.vars files in the blueprint folder. E.g --vars-file=./file.vars")

	return runCmd
}

func newRunCmdFunc(e jumppad.Engine, dt cclients.ContainerTasks, bp getter.Getter, hc http.HTTP, bc system.System, cc connector.Connector, noOpen *bool, force *bool, runVersion *string, autoApprove *bool, variables *[]string, variablesFile *string, l logger.Logger) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// create the shipyard and sub folders in the users home directory
		utils.CreateFolders()

		if *force {
			bp.SetForce(true)
			dt.SetForce(true)
		}

		// parse the vars into a map
		vars := map[string]string{}
		for _, v := range *variables {
			// if the variable is wrapped in single quotes remove them
			v = strings.TrimPrefix(v, "'")
			v = strings.TrimSuffix(v, "'")

			parts := strings.Split(v, "=")
			if len(parts) >= 2 {
				vars[parts[0]] = strings.Join(parts[1:], "=")
			}
		}

		// check the variables file exists
		if variablesFile != nil && *variablesFile != "" {
			if _, err := os.Stat(*variablesFile); err != nil {
				return fmt.Errorf("variables file %s, does not exist", *variablesFile)
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
				return fmt.Errorf("unable to generate connector certificates: %s", err)
			}
		}

		// start the connector
		if !cc.IsRunning() {
			cb, err := cc.GetLocalCertBundle(utils.CertsDir(""))
			if err != nil {
				return fmt.Errorf("unable to get certificates to secure ingress: %s", err)
			}

			l.Debug("Starting API server")

			err = cc.Start(cb)
			if err != nil {
				return fmt.Errorf("unable to start API server: %s", err)
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
			cmd.Println("Running configuration from ", dst, " -- press ctrl c to cancel")
			cmd.Println("")

			if !utils.IsLocalFolder(dst) && !utils.IsHCLFile(dst) {
				// fetch the remote server from github
				err := bp.Get(dst, utils.BlueprintLocalFolder(dst))
				if err != nil {
					return fmt.Errorf("unable to retrieve blueprint: %s", err)
				}

				dst = utils.BlueprintLocalFolder(dst)
			}
		}

		// update status every 30s to let people know we are still running
		statusUpdate := time.NewTicker(15 * time.Second)
		startTime := time.Now()

		go func() {
			for range statusUpdate.C {
				elapsedTime := time.Since(startTime).Seconds()
				l.Info(fmt.Sprintf("Please wait, still creating resources [Elapsed Time: %f]", elapsedTime))
			}
		}()

		// trap ctrl c
		done := make(chan os.Signal, 1)
		signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			<-done // Will block here until user hits ctrl+c

			// cancel the context
			cancel()
		}()

		config, err := e.ApplyWithVariables(ctx, dst, vars, *variablesFile)
		if err != nil {
			return err
		}

		// do not open the browser windows
		if !*noOpen {

			browserList := []string{}
			checkDuration := 30 * time.Second

			for _, r := range config.Resources {
				switch v := r.(type) {
				case *container.Container:
					for _, p := range v.Ports {
						if p.Host != "" && p.OpenInBrowser != "" {
							browserList = append(browserList, buildBrowserPath(r.Metadata().Name, p.Host, r.Metadata().Type, p.OpenInBrowser))
						}
					}
				case *ingress.Ingress:
					if v.OpenInBrowser != "" {
						browserList = append(browserList, buildBrowserPath(r.Metadata().Name, fmt.Sprintf("%d", v.Port), r.Metadata().Type, v.OpenInBrowser))
					}
				case *nomad.NomadCluster:
					if v.OpenInBrowser {
						// get the API port
						browserList = append(browserList, buildBrowserPath("server."+r.Metadata().Name, fmt.Sprintf("%d", v.APIPort), r.Metadata().Type, "/"))
					}
				case *docs.Docs:
					if v.OpenInBrowser {
						port := strconv.Itoa(v.Port)
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
					err := hc.HealthCheckHTTP(uri, "", map[string][]string{}, "", []int{200}, checkDuration)
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
		var b *blueprint.Blueprint
		bps, _ := config.FindResourcesByType(blueprint.TypeBlueprint)
		for _, bp := range bps {
			// pick the first blueprint in the root
			if bp.Metadata().Module == "" {
				b = bp.(*blueprint.Blueprint)
				break
			}
		}

		if b != nil {
			cmd.Println("")
			cmd.Println("########################################################")
			cmd.Println("")
			cmd.Println("Title", b.Title)
			cmd.Println("Author", b.Author)

			// parse the body as markdown and print
			intro := markdown.Render(b.Description, 80, 0)

			cmd.Println("")
			cmd.Print(string(intro))

			outputs := []*resources.Output{}
			os, _ := e.Config().FindResourcesByType(resources.TypeOutput)
			for _, o := range os {
				// only grab the root outputs
				if o.Metadata().Module == "" {
					outputs = append(outputs, o.(*resources.Output))
				}
			}

			if len(outputs) > 0 {
				cmd.Println("")
				cmd.Printf("This blueprint defines %d output variables.\n", len(outputs))
				cmd.Println("")

				maxLen := 0
				for _, o := range outputs {
					if len(o.Meta.Name) > maxLen {
						maxLen = len(o.Meta.Name)
					}
				}

				format := fmt.Sprintf(" * %%%ds: %%s\n", maxLen)

				for _, o := range outputs {
					fmt.Printf(format, o.Meta.Name, o.Value)
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
