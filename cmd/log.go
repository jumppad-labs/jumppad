package cmd

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/docker/docker/api/types"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/shipyard"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"
)

func newLogCmd(engine shipyard.Engine, dc clients.Docker) *cobra.Command {
	logCmd := &cobra.Command{
		Use:   "log <command> ",
		Short: "Tails logs for running shipyard resources",
		Long:  "Tails logs for running shipyard resources",
		Example: `
  # Tail logs for all running resources
	shipyard log

	# Tail logs for a specific resource
	shipyard log container.nginx

	# Tail logs for a kubernetes pod or deployment
	shipyard logs k8s_cluster.dev deployment/nginx
	`,
		Args: cobra.ArbitraryArgs,
		RunE: newLogCmdFunc(dc),
	}

	return logCmd
}

func newLogCmdFunc(dc clients.Docker) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		log := hclog.Default()

		// get the list of resources that can be logged
		c := config.New()
		err := c.FromJSON(utils.StatePath())
		if err != nil {
			fmt.Println("Unable to load state", err)
			os.Exit(1)
		}

		loggable := []config.Resource{}
		for _, r := range c.Resources {
			switch r.Info().Type {
			case config.TypeContainer:
				if !r.Info().Disabled {
					loggable = append(loggable, r)
				}
			case config.TypeImageCache:
				loggable = append(loggable, r)
			}
		}

		ctx := context.Background()

		for _, r := range loggable {
			rc, err := dc.ContainerLogs(
				ctx,
				utils.FQDN(r.Info().Name, string(r.Info().Type)),
				types.ContainerLogsOptions{
					ShowStdout: true,
					ShowStderr: true,
					Follow:     true,
					Tail:       "40",
				},
			)

			if err == nil {
				go copyLogOutput(rc, log)
			} else {
				log.Error("Unable to get logs for container", "error", err)
			}
		}

		// block until a signal is received
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, os.Interrupt)
		<-sigs

		return nil
	}
}

func copyLogOutput(rc io.ReadCloser, log hclog.Logger) {
	hdr := make([]byte, 8)
	for {
		_, err := rc.Read(hdr)
		if err != nil {
			log.Error("Unable to read from log stream", "error", err)
		}

		var w io.Writer
		switch hdr[0] {
		case 1:
			w = os.Stdout
		default:
			w = os.Stderr
		}
		count := binary.BigEndian.Uint32(hdr[4:])
		dat := make([]byte, count)
		_, err = rc.Read(dat)
		fmt.Fprint(w, string(dat))
	}
}
