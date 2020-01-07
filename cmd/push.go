package cmd

import (
	"github.com/spf13/cobra"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/providers"
	"fmt"
	"os"
	"strings"
)

var pushCmd = &cobra.Command{
	Use: "push [image] [cluster] [network]",
	Short: "Push a local Docker image to a cluster",
	Long:  `Push a local Docker image to a cluster`,
	Example: `yard push nicholasjackson/fake-service:v0.1.3 k3s cloud`,
	DisableFlagsInUseLine: true,
	Args: cobra.MaximumNArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		// TODO this needs validation

		image := args[0]
		cluster := args[1]
		network := args[2]

		fmt.Printf("Pushing image %s to cluster %s\n\n", image, cluster)

		pc := &config.Cluster{
			Name: cluster,
			Driver: "k3s",
			NetworkRef: &config.Network{Name: network},
		}

		dc,err := clients.NewDocker()
		if err != nil {
			fmt.Println("Error pushing image: ", err)
			os.Exit(1)
		}

		p:= providers.NewCluster(
			pc, 
			dc,
			nil,
			createLogger(),
		)

		err = p.ImportLocalDockerImages([]config.Image{config.Image{Name: strings.Trim(image, " ")}})
		if err != nil {
			fmt.Println("Error pushing image: ", err)
			os.Exit(1)
		}
	},
}