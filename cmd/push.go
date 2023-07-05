package cmd

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/hclconfig"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/jumppad-labs/jumppad/pkg/providers"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
)

func newPushCmd(ct clients.ContainerTasks, kc clients.Kubernetes, ht clients.HTTP, nc clients.Nomad, l hclog.Logger) *cobra.Command {
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
				ct.SetForcePull(true)
			}

			image := args[0]
			cluster := args[1]

			fmt.Printf("Pushing image %s to cluster %s\n\n", image, cluster)

			// check the resource is of the allowed type
			if !strings.HasPrefix(cluster, "nomad_cluster") && !strings.HasPrefix(cluster, "k8s_cluster") {
				return xerrors.Errorf("Invalid resource type, only resources type nomad_cluster and k8s_cluster are supported")
			}

			// find the cluster in the state
			p := hclconfig.NewParser(hclconfig.DefaultOptions())
			d, err := ioutil.ReadFile(utils.StatePath())
			if err != nil {
				return fmt.Errorf("Unable to read state file")
			}

			cfg, err := p.UnmarshalJSON(d)
			if err != nil {
				return fmt.Errorf("Unable to unmarshal state file")
			}

			r, err := cfg.FindResource(cluster)
			if err != nil {
				return xerrors.Errorf("Cluster %s is not running", cluster)
			}

			switch r.Metadata().Type {
			case resources.TypeK8sCluster:
				return pushK8sCluster(image, r.(*resources.K8sCluster), ct, kc, ht, l, true)
			case resources.TypeNomadCluster:
				return pushNomadCluster(image, r.(*resources.NomadCluster), ct, nc, l, true)
			}

			return nil
		},
	}

	pushCmd.Flags().BoolVarP(&force, "force-update", "", false, "When set to true jumppad will ignore cached images or files and will download all resources")

	return pushCmd
}

func pushK8sCluster(image string, c *resources.K8sCluster, ct clients.ContainerTasks, kc clients.Kubernetes, ht clients.HTTP, log hclog.Logger, force bool) error {
	cl := providers.NewK8sCluster(c, ct, kc, ht, nil, log)

	// get the id of the cluster
	ids, err := cl.Lookup()
	if err != nil {
		return xerrors.Errorf("Error getting id for cluster")
	}

	for _, id := range ids {
		log.Info("Pushing to container", "id", id, "image", image)
		err = cl.ImportLocalDockerImages(utils.ImageVolumeName, id, []resources.Image{resources.Image{Name: strings.Trim(image, " ")}}, force)
		if err != nil {
			return xerrors.Errorf("Error pushing image: %w ", err)
		}
	}

	return nil
}

func pushNomadCluster(image string, c *resources.NomadCluster, ct clients.ContainerTasks, ht clients.Nomad, log hclog.Logger, force bool) error {
	cl := providers.NewNomadCluster(c, ct, ht, nil, log)

	// get the id of the cluster
	ids, err := cl.Lookup()
	if err != nil {
		return xerrors.Errorf("Error getting id for cluster")
	}

	for _, id := range ids {
		log.Info("Pushing to container", "id", id, "image", image)
		err = cl.ImportLocalDockerImages(utils.ImageVolumeName, id, []resources.Image{resources.Image{Name: strings.Trim(image, " ")}}, force)
		if err != nil {
			return xerrors.Errorf("Error pushing image: %w ", err)
		}
	}

	return nil
}
