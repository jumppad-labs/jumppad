package cmd

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	
	"github.com/docker/docker/api/types"
	"github.com/fatih/color"
	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"
	
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/shipyard"
	"github.com/shipyard-run/shipyard/pkg/utils"
)

func newLogCmd(engine shipyard.Engine, dc clients.Docker, stdout, stderr io.Writer) *cobra.Command {
	logCmd := &cobra.Command{
		Use:     "log <command> ",
		Short:   "Tails logs for running shipyard resources",
		Long:    "Tails logs for running shipyard resources",
		Aliases: []string{"logs"},
		Example: `
  # Tail logs for all running resources
	shipyard log

	# Tail logs for a specific resource
	shipyard log container.nginx
	`,
		Args: cobra.ArbitraryArgs,
		ValidArgsFunction: getResources,
		RunE: newLogCmdFunc(dc, stdout, stderr),
	}

	return logCmd
}

var termColors = []color.Attribute{
	color.FgRed,
	color.FgGreen,
	color.FgYellow,
	color.FgBlue,
	color.FgMagenta,
	color.FgCyan,
	color.FgWhite,
}

func getResources(cmd *cobra.Command, args []string, complete string) ([]string, cobra.ShellCompDirective) {
	loggable, err := getLoggable(args)
	if err != nil {
		return []string{err.Error()}, cobra.ShellCompDirectiveNoFileComp
	}
	
	// all containers from all loggable resources
	var allContainers []string
	
	for _, r := range loggable {
		if r.Info().Disabled {
			continue
		}
		var containers []string
		// override the name for certain resources
		switch r.Info().Type {
		case config.TypeContainer:
			containers = append(containers, "container."+r.Info().Name)
		case config.TypeK8sCluster:
			containers = append(containers, "server."+r.Info().Name)
		case config.TypeNomadCluster:
			containers = append(containers, "server."+r.Info().Name)
			// add the client nodes
			nomad := r.(*config.NomadCluster)
			for n := 0; n < nomad.ClientNodes; n++ {
				containers = append(containers, fmt.Sprintf("%d.client.%s", n+1, r.Info().Name))
			}
		default:
			// no need to log these, right?
			// containers = append(containers, r.Info().Name)
		}
		allContainers = append(allContainers, containers...)
	}
	return allContainers, cobra.ShellCompDirectiveNoFileComp
	
}

func newLogCmdFunc(dc clients.Docker, stdout, stderr io.Writer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		log := hclog.Default()
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, os.Interrupt)
		waitGroup := sync.WaitGroup{}
		
		loggable, err := getLoggable(args)
		if err != nil {
			return err
		}
		
		ctx := context.Background()
		
		for _, r := range loggable {
			if r.Info().Disabled {
				continue
			}
			
			// resources can contain more than one container
			containers := []string{}
			
			// override the name for certain resources
			switch r.Info().Type {
			case config.TypeK8sCluster:
				containers = append(containers, "server."+r.Info().Name)
			case config.TypeNomadCluster:
				containers = append(containers, "server."+r.Info().Name)
				
				// add the client nodes
				nomad := r.(*config.NomadCluster)
				for n := 0; n < nomad.ClientNodes; n++ {
					containers = append(containers, fmt.Sprintf("%d.client.%s", n+1, r.Info().Name))
				}
			
			default:
				containers = append(containers, r.Info().Name)
			}
			
			for _, container := range containers {
				rc, err := dc.ContainerLogs(
					ctx,
					utils.FQDN(container, string(r.Info().Type)),
					types.ContainerLogsOptions{
						ShowStdout: true,
						ShowStderr: true,
						Follow:     true,
						Tail:       "40",
					},
				)
				
				if err == nil {
					waitGroup.Add(1)
					go func(rc io.ReadCloser, name string, c color.Attribute, log hclog.Logger) {
						writeLogOutput(rc, stdout, stderr, name, c, log)
						waitGroup.Done()
					}(rc, container, getRandomColor(), log)
				} else {
					log.Error("Unable to get logs for container", "error", err)
				}
			}
		}
		
		// send an interrupt when the waitGroup is done
		go func() {
			waitGroup.Wait()
			log.Info("No more logs to tail")
			sigs <- os.Interrupt
		}()
		
		// block until a signal is received
		<-sigs
		
		return nil
	}
}
// if this methods returns and error, it will get returned as shell-completion data
// otherwise fmt.println() gets lost
func getLoggable(args []string) ([]config.Resource, error) {
	// get the list of resources that can be logged
	c := config.New()
	err := c.FromJSON(utils.StatePath())
	if err != nil {
		return nil, fmt.Errorf("unable to load state file, check you have running resources: %s", err)
	}
	
	// if an argument is provided, only tail logs for that resource
	// first validate that the resource exists
	resources := c.Resources
	if len(args) > 0 {
		r, err := c.FindResource(args[0])
		if err != nil {
			return nil, fmt.Errorf("unable to find resource: %s", err)
		}
		
		resources = []config.Resource{r}
	}
	
	loggable := []config.Resource{}
	for _, r := range resources {
		switch r.Info().Type {
		case config.TypeContainer:
			if !r.Info().Disabled {
				loggable = append(loggable, r)
			}
		case config.TypeK8sCluster:
			if !r.Info().Disabled {
				loggable = append(loggable, r)
			}
		case config.TypeNomadCluster:
			if !r.Info().Disabled {
				loggable = append(loggable, r)
			}
		case config.TypeSidecar:
			if !r.Info().Disabled {
				loggable = append(loggable, r)
			}
		case config.TypeK8sIngress:
			if !r.Info().Disabled {
				loggable = append(loggable, r)
			}
		case config.TypeNomadIngress:
			if !r.Info().Disabled {
				loggable = append(loggable, r)
			}
		case config.TypeContainerIngress:
			if !r.Info().Disabled {
				loggable = append(loggable, r)
			}
		case config.TypeImageCache:
			loggable = append(loggable, r)
		}
	}
	return loggable, nil
}

func getRandomColor() color.Attribute {
	return termColors[rand.Intn(len(termColors)-1)]
}

func writeLogOutput(rc io.ReadCloser, stdout, stderr io.Writer, name string, c color.Attribute, log hclog.Logger) {
	hdr := make([]byte, 8)
	colorWriter := color.New(c)

	for {
		_, err := rc.Read(hdr)
		if err != nil {
			log.Error("Unable to read from log stream", "name", name, "error", err)
			return
		}

		var w io.Writer
		switch hdr[0] {
		case 1:
			w = stdout
		default:
			w = stderr
		}

		count := binary.BigEndian.Uint32(hdr[4:])
		dat := make([]byte, count)
		_, err = rc.Read(dat)

		colorWriter.Fprintf(w, "[%s]   %s", name, string(dat))
	}
}
