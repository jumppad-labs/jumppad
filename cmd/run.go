package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	gvm "github.com/nicholasjackson/version-manager"

	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/shipyard"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"

	markdown "github.com/MichaelMure/go-term-markdown"
)

func newRunCmd(e shipyard.Engine, bp clients.Getter, hc clients.HTTP, bc clients.System, vm gvm.Versions, l hclog.Logger) *cobra.Command {
	var noOpen bool
	var force bool
	var y bool
	var runVersion string
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
		RunE:         newRunCmdFunc(e, bp, hc, bc, vm, &noOpen, &force, &runVersion, &y, l),
		SilenceUsage: true,
	}

	runCmd.Flags().StringVarP(&runVersion, "version", "v", "", "When set, run creates the specified resources using a particular Shipyard version")
	runCmd.Flags().BoolVarP(&y, "y", "y", false, "When set, Shipyard will not prompt for conifirmation")
	runCmd.Flags().BoolVarP(&noOpen, "no-browser", "", false, "When set to true Shipyard will not open the browser windows defined in the blueprint")
	runCmd.Flags().BoolVarP(&force, "force-update", "", false, "When set to true Shipyard ignores cached images or files and will download all resources")

	return runCmd
}

func newRunCmdFunc(e shipyard.Engine, bp clients.Getter, hc clients.HTTP, bc clients.System, vm gvm.Versions, noOpen *bool, force *bool, runVersion *string, autoApprove *bool, l hclog.Logger) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if *force == true {
			bp.SetForce(true)
			e.GetClients().ContainerTasks.SetForcePull(true)
		}

		// Check the system to see if Docker is running and everything is installed
		s, err := bc.Preflight()
		if err != nil {
			cmd.Println("")
			cmd.Println("###### SYSTEM DIAGNOSTICS ######")
			cmd.Println(s)
			return err
		}

		// are we running with a different shipyard version, if so check it is installed
		if *runVersion != "" {
			return runWithOtherVersion(*runVersion, *autoApprove, args, *force, *noOpen, cmd, vm)
		}

		// create the shipyard home and releases
		os.MkdirAll(utils.GetReleasesFolder(), os.FileMode(0755))

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
		err = e.ParseConfig(dst)
		if err != nil {
			return fmt.Errorf("Unable to read config: %s", err)
		}

		// have we already got a blueprint in the state
		blueprintExists := false
		if bluePrintInState() {
			blueprintExists = true
		}

		// check that the current shipyard version can process this blueprint
		if e.Blueprint() != nil && e.Blueprint().ShipyardVersion != "" {
			valid, err := vm.InRange(version, e.Blueprint().ShipyardVersion)
			if err != nil {
				return err
			}

			if !valid {
				// we neeed to go in to the check loop
				return runWithOtherVersion(e.Blueprint().ShipyardVersion, *autoApprove, args, *force, *noOpen, cmd, vm)
			}
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
			checkDuration, err := time.ParseDuration(e.Blueprint().HealthCheckTimeout)
			if err != nil {
				checkDuration = 30 * time.Second
			}

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
					err := hc.HealthCheckHTTP(uri, checkDuration)
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
				cmd.Println("######################################################")
				cmd.Println("")
				cmd.Println("Environment Variables")
				cmd.Println("")
				cmd.Println("######################################################")

				cmd.Println("")
				cmd.Println("This blueprint exports the following environment variables:")
				cmd.Println("")
				for _, env := range e.Blueprint().Environment {
					cmd.Printf("\033[1;32m%s=%s\033[0m\n", env.Key, env.Value)
				}
				cmd.Println("")
				cmd.Println("You can set exported environment variables for your current terminal session using the following command:")
				cmd.Println("")

				if runtime.GOOS == "windows" {
					cmd.Println(`@FOR /f "tokens=*" %i IN ('shipyard env') DO @%i`)
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

func runWithOtherVersion(version string, autoApprove bool, args []string, forceUpdate bool, noBrowser bool, cmd *cobra.Command, vm gvm.Versions) error {

	var exePath string

	r, err := vm.ListInstalledVersions(version)
	if err != nil {
		return err
	}

	// find a version suitable
	tag, url, err := vm.GetLatestReleaseURL(version)
	if err != nil {
		return err
	}

	if len(r) != 1 {

		// only prompt if not auto approve
		if !autoApprove {
			cmd.Print("Would you like to install version: ", tag, " [y/n]: ")

			scanner := bufio.NewScanner(cmd.InOrStdin())

			var text string
			scanner.Scan()
			text = scanner.Text()

			if text != "y" {
				return nil
			}
		}

		exePath, err = vm.DownloadRelease(tag, url)
		if err != nil {
			return err
		}
	} else {
		exePath = r[tag]
	}

	// execute shipyard using a sub process
	cmd.Println("Running blueprint with version:", tag)

	//if there is no path use the current folder
	if len(args[0]) == 0 || args[0] == "" {
		p, _ := os.Getwd()
		args[0] = p
	} else if strings.HasPrefix(args[0], ".") {
		// if we have a relative path we need to convert to an absolute path
		p, _ := filepath.Abs(args[0])
		args[0] = p
	}

	commandString := []string{
		"run",
	}

	if forceUpdate {
		commandString = append(commandString, "--force-update")
	}

	if noBrowser {
		commandString = append(commandString, "--no-browser")
	}

	commandString = append(commandString, args[0])

	execCmd := exec.Command(exePath, commandString...)
	execCmd.Stderr = cmd.ErrOrStderr()
	execCmd.Stdout = cmd.ErrOrStderr()

	err = execCmd.Start()
	if err != nil {
		return err
	}

	err = execCmd.Wait()
	if err != nil {
		return err
	}

	return nil
}
