package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restarts all paused resources for the currently active blueprint",
	Long:  `Restarts all paused resources for the currently active blueprint`,
	Example: `
  shipyard start
	`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		fmt.Println("Restarting resources")

		// create a docker client
		c, err := clients.NewDocker()
		if err != nil {
			fmt.Println("Unable to connect to Docker daemon", err)
			os.Exit(1)
		}

		filters := filters.NewArgs()
		filters.Add("name", "shipyard")
		filters.Add("status", "exited")

		cl, err := c.ContainerList(
			context.Background(),
			types.ContainerListOptions{
				Filters: filters,
			},
		)

		for _, con := range cl {
			err := c.ContainerStart(context.Background(), con.ID, types.ContainerStartOptions{})
			if err != nil {
				fmt.Println("Unable to start container", con.Names[0], err)
				os.Exit(1)
			}
		}
	},
}
