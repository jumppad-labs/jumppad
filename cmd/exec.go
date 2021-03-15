package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/docker/docker/pkg/term"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
)

func newExecCmd(dt clients.ContainerTasks) *cobra.Command {
	return &cobra.Command{
		Use:   "exec <resource> <pod> <container> -- <command>",
		Short: "Execute a command in a Resource",
		Long:  `Execute a command in a Resource or start a Tools resource and execute`,
		Example: `
		# Execute a command in the first container of a Kubernetes pod
		shipyard exec k8s_cluster.k3s mypod -- ls -las
		
		# Execute a command in the named container of a Kubernetes pod
		shipyard exec k8s_cluster.k3s mypod web -- ls -las

		# Create a bash shell in a container
		shipyard exec container.consul -- bash
		
		# Create a default shell in a container
		shipyard exec container.consul
		`,
		Args:               cobra.MinimumNArgs(1),
		DisableFlagParsing: true,
		SilenceUsage:       true,
		RunE: func(cmd *cobra.Command, args []string) error {
			parameters, command := parseParameters(args)

			// find a list of resources in the current stack
			sc := config.New(true)
			err := sc.FromJSON(utils.StatePath())
			if err != nil {
				return fmt.Errorf("No resources are running, start a stack with 'shipyard run [blueprint]'")
			}

			// get the resource
			r, err := sc.FindResource(parameters[0])
			if err != nil {
				return xerrors.Errorf("Unable to find resource %s: %w", parameters[0], err)
			}

			switch r.Info().Type {
			case config.TypeContainer:
				return createContainerShell(r, dt, command)
			case config.TypeK8sCluster:
				pod := ""
				container := ""

				if len(parameters) != 2 {
					return fmt.Errorf("Please specify a Kubernetes pod or service for this cluster")
				}

				// no pod specified use default
				if len(parameters) == 2 {
					pod = parameters[1]
				}

				if len(parameters) == 3 {
					pod = parameters[1]
					container = parameters[2]
				}

				return createK8sShell(r, dt, pod, container, command)
			case config.TypeNomadCluster:
			default:
				return fmt.Errorf("Unknown resource type")
			}

			return nil
		},
	}
}

// parse parameters splits the args from the command to be executed
func parseParameters(args []string) ([]string, []string) {
	commandIndex := -1
	for p, v := range args {
		if v == "--" {
			commandIndex = p
			break
		}
	}

	if commandIndex == -1 {
		return args, []string{}
	}

	return args[0:commandIndex], args[commandIndex+1:]
}

func createContainerShell(r config.Resource, dt clients.ContainerTasks, command []string) error {
	if len(command) == 0 {
		command = []string{"sh"}
	}

	// find the container id
	ids, err := dt.FindContainerIDs(r.Info().Name, config.TypeContainer)
	if err != nil || len(ids) == 0 {
		return fmt.Errorf("Unable to find container %s", r.Info().Name)
	}

	in, stdout, _ := term.StdStreams()
	err = dt.CreateShell(ids[0], command, in, stdout, stdout)
	if err != nil {
		return fmt.Errorf("Could not execute command for container %s. Error: %s", ids[0], err)
	}

	return nil
}

func createK8sShell(r config.Resource, dt clients.ContainerTasks, pod, container string, command []string) error {
	clusterName := r.Info().Name

	exec := []string{"kubectl", "exec", "-ti", pod}

	if container != "" {
		exec = append(exec, "-c", container)
	}

	if len(command) == 0 {
		command = []string{"sh"}
	}

	// start a tools container
	i := config.Image{Name: "shipyardrun/ingress:latest"}
	err := dt.PullImage(i, false)
	if err != nil {
		return xerrors.Errorf("Could pull ingress image. Error: %w", err)
	}

	// create the new container for the exec and add it to the config
	c := config.NewContainer(fmt.Sprintf("exec-%d", time.Now().Nanosecond()))
	r.AddChild(c)

	c.Image = &i
	c.Entrypoint = []string{} // overide the entrypoint
	c.Command = []string{"tail", "-f", "/dev/null"}

	c.Networks = r.(*config.K8sCluster).Networks

	wd, err := os.Getwd()
	if err != nil {
		return xerrors.Errorf("Could not get working directory. Error: %w", err)
	}

	c.Volumes = []config.Volume{
		config.Volume{
			Source:      wd,
			Destination: "/files",
		},
		config.Volume{
			Source:      utils.ShipyardHome(),
			Destination: "/root/.shipyard",
		},
	}

	c.Environment = []config.KV{
		config.KV{
			Key:   "KUBECONFIG",
			Value: fmt.Sprintf("/root/.shipyard/config/%s/kubeconfig-docker.yaml", clusterName),
		},
	}

	tools, err := dt.CreateContainer(c)
	if err != nil {
		return fmt.Errorf("Could not create exec container. Error: %s", err)
	}
	defer dt.RemoveContainer(tools)

	in, stdout, _ := term.StdStreams()
	err = dt.CreateShell(tools, append(exec, command...), in, stdout, stdout)
	if err != nil {
		return fmt.Errorf("Could not execute command for cluster %s. Error: %s", clusterName, err)
	}

	return nil
}
