package cmd

import (
	"fmt"
	"os"

	"github.com/hokaccha/go-prettyjson"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"
)

/*
[ CREATED ] network.cloud (green)
[ FAILED  ] k8s_cluster.k3s (red)
[ PENDING ] helm.vault (gray)
*/

const (
	Black   = "\033[1;30m%s\033[0m"
	Red     = "\033[1;31m%s\033[0m"
	Green   = "\033[1;32m%s\033[0m"
	Yellow  = "\033[1;33m%s\033[0m"
	Purple  = "\033[1;34m%s\033[0m"
	Magenta = "\033[1;35m%s\033[0m"
	Teal    = "\033[1;36m%s\033[0m"
	White   = "\033[1;37m%s\033[0m"
)

var json bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of the current stack",
	Long:  `Show the status of the current stack`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// load the stack
		c := config.New()
		err := c.FromJSON(utils.StatePath())
		if err != nil {
			fmt.Println("Unable to load state", err)
			os.Exit(1)
		}

		if json {
			s, err := prettyjson.Marshal(c)
			if err != nil {
				fmt.Println("Unable to load state", err)
				os.Exit(1)
			}

			fmt.Println(string(s))
		} else {

			createdCount := 0
			failedCount := 0
			pendingCount := 0

			fmt.Println()
			for _, r := range c.Resources {
				status := fmt.Sprintf(White, "PENDING")
				switch r.Info().Status {
				case config.Applied:
					status = fmt.Sprintf(Green, "CREATED")
					createdCount++
				case config.Failed:
					status = fmt.Sprintf(Red, "FAILED")
					failedCount++
				default:
					pendingCount++
				}
				fmt.Printf(" [ %s ] %s.%s\n", status, r.Info().Type, r.Info().Name)
			}

			fmt.Println()
			fmt.Printf("Pending: %d Created: %d Failed: %d\n", pendingCount, createdCount, failedCount)
		}
	},
}

func init() {
	statusCmd.Flags().BoolVarP(&json, "json", "", false, "Output the status as JSON")
}
