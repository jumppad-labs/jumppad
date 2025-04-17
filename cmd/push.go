package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/instruqt/jumppad/pkg/clients"
	"github.com/instruqt/jumppad/pkg/clients/container"
	"github.com/instruqt/jumppad/pkg/clients/container/types"
	"github.com/instruqt/jumppad/pkg/clients/logger"
	"github.com/instruqt/jumppad/pkg/config"
	"github.com/instruqt/jumppad/pkg/config/resources/k8s"
	"github.com/instruqt/jumppad/pkg/config/resources/nomad"
	"github.com/spf13/cobra"
)

func newPushCmd(ct container.ContainerTasks, l logger.Logger) *cobra.Command {
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
				return errors.New("push requires two arguments [image] [cluster]")
			}

			if force {
				ct.SetForce(true)
			}

			image := args[0]
			cluster := args[1]

			fmt.Printf("Pushing image %s to cluster %s\n\n", image, cluster)

			// check the resource is of the allowed type
			if !strings.Contains(cluster, "nomad_cluster") && !strings.Contains(cluster, "k8s_cluster") {
				return errors.New("invalid resource type, only resources type nomad_cluster and k8s_cluster are supported")
			}

			c, err := config.LoadState()
			if err != nil {
				cmd.Println("Error: Unable to load state, ", err)
				os.Exit(1)
			}

			r, err := c.FindResource(cluster)
			if err != nil {
				return fmt.Errorf("cluster %s is not running", cluster)
			}

			switch r.Metadata().Type {
			case k8s.TypeK8sCluster:
				return pushK8sCluster(image, r.(*k8s.Cluster), l, true)
			case nomad.TypeNomadCluster:
				return pushNomadCluster(image, r.(*nomad.NomadCluster), l, true)
			}

			return nil
		},
	}

	pushCmd.Flags().BoolVarP(&force, "force-update", "", false, "When set to true jumppad will ignore cached images or files and will download all resources")

	return pushCmd
}

func pushK8sCluster(image string, c *k8s.Cluster, log logger.Logger, force bool) error {
	cli, _ := clients.GenerateClients(log)
	p := config.NewProviders(cli)
	cl := p.GetProvider(c).(*k8s.ClusterProvider)

	// get the id of the cluster
	ids, err := cl.Lookup()
	if err != nil {
		return errors.New("error getting id for cluster")
	}

	for _, id := range ids {
		log.Info("Pushing to container", "id", id, "image", image)
		err = cl.ImportLocalDockerImages([]types.Image{{Name: strings.Trim(image, " ")}}, force)
		if err != nil {
			return fmt.Errorf("error pushing image: %w ", err)
		}
	}

	return nil
}

func pushNomadCluster(image string, c *nomad.NomadCluster, log logger.Logger, force bool) error {
	cli, _ := clients.GenerateClients(log)
	p := config.NewProviders(cli)
	cl := p.GetProvider(c).(*nomad.ClusterProvider)

	// get the id of the cluster

	log.Info("Pushing to container", "ref", c.Meta.ID, "image", image)
	err := cl.ImportLocalDockerImages([]types.Image{{Name: strings.Trim(image, " ")}}, force)
	if err != nil {
		return fmt.Errorf("error pushing image: %w ", err)
	}

	return nil
}
