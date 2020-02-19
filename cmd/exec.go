package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/pkg/term"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:                "exec",
	Short:              "Execute a command in a Resource",
	Long:               `Execute a command in a Resource or start a Tools resource and execute`,
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: true,
	Run: func(cmd *cobra.Command, args []string) {
		l := createLogger()
		cd, _ := clients.NewDocker()
		dt := clients.NewDockerTasks(cd, l)

		// find a list of resources in the current stack
		sc := config.New()
		err := sc.FromJSON(utils.StatePath())
		if err != nil {
			l.Error("No resources are running, start a stack with 'shipyard run [blueprint]'")
			return
		}

		parameters, command := parseParameters(args)

		fmt.Printf("parameters: %#v - command: %#v\n", parameters, command)

		targets := strings.Split(parameters[0], ".")
		if len(targets) < 2 {
			l.Error("No target specified for resource")
			return
		}

		switch targets[0] {
		case string(config.TypeContainer):
			container := targets[1]

			if len(command) == 0 {
				command = []string{"sh"}
			}

			// find the container id
			ids, err := dt.FindContainerIDs(container, config.TypeContainer)
			if err != nil || len(ids) == 0 {
				l.Error("Unable to find container", "container", container)
				return
			}

			in, stdout, _ := term.StdStreams()
			err = dt.CreateShell(ids[0], command, in, stdout, stdout)
			if err != nil {
				l.Error("Could not execute command", "container", ids[0], "error", err)
				return
			}
		case string(config.TypeK8sCluster):
			// shipyard exec k8s_cluster.k3s <pod> -- <command>
			clusterName := targets[1]

			// check if the given cluster exists
			cluster, err := sc.FindResource(fmt.Sprintf("%s.%s", config.TypeK8sCluster, clusterName))
			if err != nil {
				l.Error("Unable to find cluster", "cluster", clusterName, "error", err)
				return
			}

			exec := []string{"kubectl", "exec", "-ti"}

			// get the pod to execute the command in
			if len(parameters) == 1 {
				l.Error("No target specified", "cluster", clusterName)
				return
			}

			exec = append(exec, parameters[1])

			if len(parameters) == 3 {
				exec = append(exec, "-c", parameters[2])
			}

			if len(command) == 0 {
				command = []string{"sh"}
			}

			// start a tools container
			i := config.Image{Name: "shipyardrun/tools:latest"}
			err = dt.PullImage(i, false)
			if err != nil {
				l.Error("Could pull tools image", "error", err)
				return
			}

			c := config.NewContainer(fmt.Sprintf("exec-%d", time.Now().Nanosecond()))
			sc.AddResource(c)
			c.Image = i
			c.Command = []string{"tail", "-f", "/dev/null"}

			c.Networks = cluster.(*config.K8sCluster).Networks

			wd, err := os.Getwd()
			if err != nil {
				l.Error("Could not get working directory", "error", err)
				return
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
				l.Error("Could not create tools container", "error", err)
				return
			}
			defer dt.RemoveContainer(tools)

			in, stdout, _ := term.StdStreams()
			err = dt.CreateShell(tools, append(exec, command...), in, stdout, stdout)
			if err != nil {
				l.Error("Could not execute command", "cluster", clusterName, "error", err)
				return
			}
		case string(config.TypeNomadCluster):
		default:
			l.Error("Unknown resource type")
			os.Exit(1)
		}
	},
}

func parseParameters(args []string) ([]string, []string) {
	commandIndex := -1
	for p, v := range args {
		if v == "--" {
			commandIndex = p
		}
	}

	if commandIndex == -1 {
		return args, []string{}
	}

	return args[0:commandIndex], args[commandIndex:]
}
