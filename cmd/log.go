package cmd

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/fatih/color"
	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"

	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/jumppad-labs/jumppad/pkg/shipyard"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/shipyard-run/hclconfig"
)

func newLogCmd(engine shipyard.Engine, dc clients.Docker, stdout, stderr io.Writer) *cobra.Command {
	logCmd := &cobra.Command{
		Use:     "log <command> ",
		Short:   "Tails logs for running shipyard resources",
		Long:    "Tails logs for running shipyard resources",
		Aliases: []string{"logs"},
		Example: `
  # Tail logs for all running resources
	jumppad log

	# Tail logs for a specific resource
	jumppad log container.nginx
	`,
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: getResources,
		RunE:              newLogCmdFunc(dc, stdout, stderr),
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
	loggable, err := getLoggable()
	if err != nil {
		return []string{err.Error()}, cobra.ShellCompDirectiveNoFileComp
	}

	return loggable, cobra.ShellCompDirectiveNoFileComp
}

func newLogCmdFunc(dc clients.Docker, stdout, stderr io.Writer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		log := hclog.Default()
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, os.Interrupt)
		waitGroup := sync.WaitGroup{}

		var loggable []string

		if len(args) == 1 {
			loggable = []string{args[0]}
		} else {
			var err error
			loggable, err = getLoggable()
			if err != nil {
				return err
			}
		}

		ctx := context.Background()

		for _, r := range loggable {
			rc, err := dc.ContainerLogs(
				ctx,
				r,
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
				}(rc, r, getRandomColor(), log)
			} else {
				log.Error("Unable to get logs for container", "error", err)
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
func getLoggable() ([]string, error) {
	// get the list of resources that can be logged
	p := hclconfig.NewParser(hclconfig.DefaultOptions())
	d, err := ioutil.ReadFile(utils.StatePath())
	if err != nil {
		return nil, fmt.Errorf("Unable to read state file")
	}

	cfg, err := p.UnmarshalJSON(d)
	if err != nil {
		return nil, fmt.Errorf("Unable to unmarshal state file")
	}

	// if an argument is provided, only tail logs for that resource
	// first validate that the resource exists

	loggable := []string{}
	for _, r := range cfg.Resources {
		if r.Metadata().Disabled {
			continue
		}

		switch r.Metadata().Type {
		case resources.TypeContainer:
			loggable = append(loggable, utils.FQDN(r.Metadata().Name, r.Metadata().Module, r.Metadata().Type))
		case resources.TypeK8sCluster:
			loggable = append(loggable, fmt.Sprintf("%s.%s", "server", utils.FQDN(r.Metadata().Name, r.Metadata().Module, r.Metadata().Type)))
		case resources.TypeNomadCluster:
			loggable = append(loggable, fmt.Sprintf("%s.%s", "server", utils.FQDN(r.Metadata().Name, r.Metadata().Module, r.Metadata().Type)))

			// add the client nodes
			nomad := r.(*resources.NomadCluster)
			for n := 0; n < nomad.ClientNodes; n++ {
				loggable = append(loggable, fmt.Sprintf("%d.%s.%s", n+1, "client", utils.FQDN(r.Metadata().Name, r.Metadata().Module, r.Metadata().Type)))
			}
		case resources.TypeSidecar:
			fallthrough
		case resources.TypeImageCache:
			loggable = append(loggable, utils.FQDN(r.Metadata().Name, r.Metadata().Module, r.Metadata().Type))
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

		name = strings.TrimSuffix(name, ".jumppad.dev")
		colorWriter.Fprintf(w, "[%s]   %s", name, string(dat))
	}
}
