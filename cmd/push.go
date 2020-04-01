package cmd

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/providers"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
)

func newPushCmd(ct clients.ContainerTasks, kc clients.Kubernetes, ht clients.HTTP, nc clients.Nomad, l hclog.Logger) *cobra.Command {
	return &cobra.Command{
		Use:                   "push [image] [cluster]",
		Short:                 "Push a local Docker image to a cluster",
		Long:                  `Push a local Docker image to a cluster`,
		Example:               `yard push nicholasjackson/fake-service:v0.1.3 k8s_cluster.k3s`,
		DisableFlagsInUseLine: true,
		Args:                  cobra.MaximumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return xerrors.Errorf("Push requires two arguments [image] [cluster]")
			}

			image := args[0]
			cluster := args[1]

			fmt.Printf("Pushing image %s to cluster %s\n\n", image, cluster)

			// check the resource is of the allowed type
			if !strings.HasPrefix(cluster, "nomad_cluster") && !strings.HasPrefix(cluster, "k8s_cluster") {
				return xerrors.Errorf("Invalid resoruce type, only resources type nomad_cluster and k8s_cluster are supported")
			}

			// find the cluster in the state
			sc := config.New()
			err := sc.FromJSON(utils.StatePath())
			if err != nil {
				return xerrors.Errorf("No resources are running, start a stack with 'shipyard run [blueprint]'")
			}

			p, err := sc.FindResource(cluster)
			if err != nil {
				return xerrors.Errorf("Cluster %s is not running", cluster)
			}

			switch p.Info().Type {
			case config.TypeK8sCluster:
				return pushK8sCluster(image, p.(*config.K8sCluster), ct, kc, ht, l)
			case config.TypeNomadCluster:
				return pushNomadCluster(image, p.(*config.NomadCluster), ct, nc, l)
			}

			return nil
		},
	}
}

func pushK8sCluster(image string, c *config.K8sCluster, ct clients.ContainerTasks, kc clients.Kubernetes, ht clients.HTTP, log hclog.Logger) error {
	cl := providers.NewK8sCluster(c, ct, kc, ht, log)

	// get the id of the cluster
	ids, err := cl.Lookup()
	if err != nil {
		return xerrors.Errorf("Error getting id for cluster")
	}

	for _, id := range ids {
		log.Info("Pushing to container", "id", id, "image", image)
		err = cl.ImportLocalDockerImages(c.Name, id, []config.Image{config.Image{Name: strings.Trim(image, " ")}})
		if err != nil {
			return xerrors.Errorf("Error pushing image: %w ", err)
		}
	}

	return nil
}

func pushNomadCluster(image string, c *config.NomadCluster, ct clients.ContainerTasks, ht clients.Nomad, log hclog.Logger) error {
	cl := providers.NewNomadCluster(c, ct, ht, log)

	// get the id of the cluster
	ids, err := cl.Lookup()
	if err != nil {
		return xerrors.Errorf("Error getting id for cluster")
	}

	for _, id := range ids {
		log.Info("Pushing to container", "id", id, "image", image)
		err = cl.ImportLocalDockerImages(c.Name, id, []config.Image{config.Image{Name: strings.Trim(image, " ")}})
		if err != nil {
			return xerrors.Errorf("Error pushing image: %w ", err)
		}
	}

	return nil
}
