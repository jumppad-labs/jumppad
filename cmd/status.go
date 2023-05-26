package cmd

import (
	"fmt"
	"os"

	"github.com/hokaccha/go-prettyjson"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/jumppad-labs/jumppad/pkg/jumppad/constants"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/shipyard-run/hclconfig/types"
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

var jsonFlag bool
var resourceType string

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of the current resources",
	Long:  `Show the status of the current resources`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// load the resources from state

		cfg, err := resources.LoadState()
		if err != nil {
			fmt.Printf("Unable to read state file")
			os.Exit(1)
		}

		if jsonFlag {
			s, err := prettyjson.Marshal(cfg)
			if err != nil {
				fmt.Println("Unable to output state as JSON", err)
				os.Exit(1)
			}

			fmt.Println(string(s))
		} else {
			fmt.Println()
			fmt.Printf("%-13s %-60s %s\n", "STATUS", "RESOURCE", "FQDN")

			createdCount := 0
			failedCount := 0
			disabledCount := 0
			pendingCount := 0

			// sort the resources
			resourceMap := map[string][]types.Resource{}

			for _, r := range cfg.Resources {
				if resourceMap[r.Metadata().Type] == nil {
					resourceMap[r.Metadata().Type] = []types.Resource{}
				}

				resourceMap[r.Metadata().Type] = append(resourceMap[r.Metadata().Type], r)
			}

			for _, ress := range resourceMap {
				for _, r := range ress {
					if (resourceType != "" && r.Metadata().Type != resourceType) || r.Metadata().Type == types.TypeModule {
						continue
					}

					status := fmt.Sprintf(White, "[ PENDING ]  ")
					if r.Metadata().Disabled {
						status = fmt.Sprintf(Teal, "[ DISABLED ] ")
						disabledCount++
					} else {
						switch r.Metadata().Properties[constants.PropertyStatus] {
						case constants.StatusCreated:
							status = fmt.Sprintf(Green, "[ CREATED ]  ")
							createdCount++
						case constants.StatusFailed:
							status = fmt.Sprintf(Red, "[ FAILED ]   ")
							failedCount++
						default:
							pendingCount++
						}
					}

					switch r.Metadata().Type {
					case resources.TypeNomadCluster:
						fmt.Printf("%-13s %-50s %s\n", status, r.Metadata().ID, fmt.Sprintf("%s.%s", "server", utils.FQDN(r.Metadata().Name, "", string(r.Metadata().Type))))

						// add the client nodes
						nomad := r.(*resources.NomadCluster)
						for n := 0; n < nomad.ClientNodes; n++ {
							fmt.Printf("%-13s %-50s %s\n", "", "", fmt.Sprintf("%d.%s.%s", n+1, "client", utils.FQDN(r.Metadata().Name, "", r.Metadata().Type)))
						}
					case resources.TypeK8sCluster:
						fmt.Printf("%-13s %-50s %s\n", status, r.Metadata().ID, fmt.Sprintf("%s.%s", "server", utils.FQDN(r.Metadata().Name, "", r.Metadata().Type)))
					case resources.TypeContainer:
						fallthrough
					case resources.TypeSidecar:
						fallthrough
					case resources.TypeImageCache:
						fmt.Printf("%-13s %-50s %s\n", status, r.Metadata().ID, "")
					default:
						fmt.Printf("%-13s %-50s %s\n", status, r.Metadata().ID, "")
					}
				}
			}

			fmt.Println()
			fmt.Printf("Pending: %d Created: %d Failed: %d Disabled: %d\n", pendingCount, createdCount, failedCount, disabledCount)
		}
	},
}

func init() {
	statusCmd.Flags().BoolVarP(&jsonFlag, "json", "", false, "Output the status as JSON")
	statusCmd.Flags().StringVarP(&resourceType, "type", "", "", "Resource type used to filter status list")
}
