package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/providers"
	"github.com/shipyard-run/shipyard/pkg/shipyard"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:                   "push [image] [cluster]",
	Short:                 "Push a local Docker image to a cluster",
	Long:                  `Push a local Docker image to a cluster`,
	Example:               `yard push nicholasjackson/fake-service:v0.1.3 k8s_cluster.k3s`,
	DisableFlagsInUseLine: true,
	Args:                  cobra.MaximumNArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		l := createLogger()

		if len(args) < 2 {
			fmt.Println("Push requires two arguments [image] [cluster]")
			os.Exit(1)
		}

		image := args[0]
		cluster := args[1]

		fmt.Printf("Pushing image %s to cluster %s\n\n", image, cluster)

		// find the cluster in the state
		sc := config.New()
		err := sc.FromJSON(utils.StatePath())
		if err != nil {
			l.Error("No resources are running, start a stack with 'shipyard run [blueprint]'")
			return
		}

		dt, _ := shipyard.GenerateClients(l)

		p, err := sc.FindResource(cluster)
		if err != nil {
			fmt.Printf("Cluster %s does not exist", cluster)
			os.Exit(1)
		}

		cl := providers.NewK8sCluster(p.(*config.K8sCluster), dt.ContainerTasks, dt.Kubernetes, dt.HTTP, l)

		// get the id of the cluster
		ids, err := cl.Lookup()
		if err != nil {
			fmt.Println("Error getting id for cluster")
			os.Exit(1)
		}

		for _, id := range ids {
			l.Info("Pushing to container", "id", id, "image", image)
			err = cl.ImportLocalDockerImages(p.Info().Name, id, []config.Image{config.Image{Name: strings.Trim(image, " ")}})
			if err != nil {
				fmt.Println("Error pushing image: ", err)
				os.Exit(1)
			}
		}
	},
}
