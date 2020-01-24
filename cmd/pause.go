package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/spf13/cobra"
)

var pauseCmd = &cobra.Command{
	Use:   "pause",
	Short: "Pauses all resources for the currently active blueprint",
	Long:  `Pause all resources for the currently active blueprint freeing up memory and CPU`,
	Example: `
  shipyard pause 
	`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		fmt.Println("Pausing resources")

		// create a docker client
		c, err := clients.NewDocker()
		if err != nil {
			fmt.Println("Unable to connect to Docker daemon", err)
			os.Exit(1)
		}

		filters := filters.NewArgs()
		filters.Add("name", "shipyard")
		filters.Add("status", "running")

		cl, err := c.ContainerList(
			context.Background(),
			types.ContainerListOptions{
				Filters: filters,
			},
		)

		sd := 20 * time.Second
		for _, con := range cl {
			err := c.ContainerStop(context.Background(), con.ID, &sd)
			if err != nil {
				fmt.Println("Unable to stop container", con.Names[0], err)
				os.Exit(1)
			}
		}
	},
}
