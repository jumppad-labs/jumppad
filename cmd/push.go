package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/container"
	"github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/http"
	ck8s "github.com/jumppad-labs/jumppad/pkg/clients/k8s"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	cnomad "github.com/jumppad-labs/jumppad/pkg/clients/nomad"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/k8s"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/nomad"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
)

func newPushCmd(ct container.ContainerTasks, kc ck8s.Kubernetes, ht http.HTTP, nc cnomad.Nomad, l logger.Logger) *cobra.Command {
	var force bool

	pushCmd := &cobra.Command{
		Use:                   "push [image] [cluster]",
		Short:                 "Push a local Docker image to a cluster",
		Long:                  `Push a local Docker image to a cluster`,
		Example:               `yard push nicholasjackson/fake-service:v0.1.3 k8s_cluster.k3s`,
		DisableFlagsInUseLine: true,
		Args:                  cobra.MaximumNArgs(3),
		SilenceUsage:          true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return xerrors.Errorf("Push requires two arguments [image] [cluster]")
			}

			if force == true {
				ct.SetForce(true)
			}

			image := args[0]
			cluster := args[1]

			fmt.Printf("Pushing image %s to cluster %s\n\n", image, cluster)

			// check the resource is of the allowed type
			if !strings.Contains(cluster, "nomad_cluster") && !strings.Contains(cluster, "k8s_cluster") {
				return xerrors.Errorf("Invalid resource type, only resources type nomad_cluster and k8s_cluster are supported")
			}

			c, err := config.LoadState()
			if err != nil {
				cmd.Println("Error: Unable to load state, ", err)
				os.Exit(1)
			}

			r, err := c.FindResource(cluster)
			if err != nil {
				return xerrors.Errorf("Cluster %s is not running", cluster)
			}

			switch r.Metadata().Type {
			case k8s.TypeK8sCluster:
				return pushK8sCluster(image, r.(*k8s.K8sCluster), ct, kc, ht, l, true)
			case nomad.TypeNomadCluster:
				return pushNomadCluster(image, r.(*nomad.NomadCluster), ct, nc, l, true)
			}

			return nil
		},
	}

	pushCmd.Flags().BoolVarP(&force, "force-update", "", false, "When set to true jumppad will ignore cached images or files and will download all resources")

	return pushCmd
}

func pushK8sCluster(image string, c *k8s.K8sCluster, ct container.ContainerTasks, kc ck8s.Kubernetes, ht http.HTTP, log logger.Logger, force bool) error {
	cli, _ := clients.GenerateClients(log)
	p := config.NewProviders(cli)
	cl := p.GetProvider(c).(*k8s.ClusterProvider)

	// get the id of the cluster
	ids, err := cl.Lookup()
	if err != nil {
		return xerrors.Errorf("Error getting id for cluster")
	}

	for _, id := range ids {
		log.Info("Pushing to container", "id", id, "image", image)
		err = cl.ImportLocalDockerImages([]types.Image{types.Image{Name: strings.Trim(image, " ")}}, force)
		if err != nil {
			return xerrors.Errorf("Error pushing image: %w ", err)
		}
	}

	return nil
}

func pushNomadCluster(image string, c *nomad.NomadCluster, ct container.ContainerTasks, ht cnomad.Nomad, log logger.Logger, force bool) error {
	cli, _ := clients.GenerateClients(log)
	p := config.NewProviders(cli)
	cl := p.GetProvider(c).(*nomad.ClusterProvider)

	// get the id of the cluster

	log.Info("Pushing to container", "ref", c.Meta.ID, "image", image)
	err := cl.ImportLocalDockerImages([]types.Image{types.Image{Name: strings.Trim(image, " ")}}, force)
	if err != nil {
		return xerrors.Errorf("Error pushing image: %w ", err)
	}

	return nil
}
