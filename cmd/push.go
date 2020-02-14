package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/providers"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:                   "push [image] [cluster] [network]",
	Short:                 "Push a local Docker image to a cluster",
	Long:                  `Push a local Docker image to a cluster`,
	Example:               `yard push nicholasjackson/fake-service:v0.1.3 k3s cloud`,
	DisableFlagsInUseLine: true,
	Args:                  cobra.MaximumNArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		// TODO this needs validation

		image := args[0]
		cluster := args[1]
		network := args[2]

		fmt.Printf("Pushing image %s to cluster %s\n\n", image, cluster)

		pc := config.NewK8sCluster(cluster)
		pc.Networks = []config.NetworkAttachment{config.NetworkAttachment{Name: network}}

		dc, err := clients.NewDocker()
		if err != nil {
			fmt.Println("Error pushing image: ", err)
			os.Exit(1)
		}

		dt := clients.NewDockerTasks(dc, hclog.Default())

		p := providers.NewK8sCluster(
			pc,
			dt,
			nil,
			nil,
			createLogger(),
		)

		// get the id of the cluster
		ids, err := p.Lookup()
		if err != nil {
			fmt.Println("Error getting id for cluster")
			os.Exit(1)
		}

		for _, id := range ids {
			err = p.ImportLocalDockerImages(cluster, id, []config.Image{config.Image{Name: strings.Trim(image, " ")}})
			if err != nil {
				fmt.Println("Error pushing image: ", err)
				os.Exit(1)
			}
		}
	},
}
