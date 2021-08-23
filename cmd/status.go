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

var jsonFlag bool
var resourceType string

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

		if jsonFlag {
			s, err := prettyjson.Marshal(c)
			if err != nil {
				fmt.Println("Unable to load state", err)
				os.Exit(1)
			}

			fmt.Println(string(s))
		} else {
			fmt.Println()
			fmt.Printf("%-13s %-30s %s\n", "STATUS", "RESOURCE", "FQDN")

			createdCount := 0
			failedCount := 0
			pendingCount := 0

			// sort the resources
			resources := map[config.ResourceType][]config.Resource{}

			for _, r := range c.Resources {
				if resources[r.Info().Type] == nil {
					resources[r.Info().Type] = []config.Resource{}
				}

				resources[r.Info().Type] = append(resources[r.Info().Type], r)
			}

			for _, ress := range resources {
				for _, r := range ress {
					if resourceType != "" && string(r.Info().Type) != resourceType {
						continue
					}

					status := fmt.Sprintf(White, "[ PENDING ]  ")
					switch r.Info().Status {
					case config.Applied:
						status = fmt.Sprintf(Green, "[ CREATED ]  ")
						createdCount++
					case config.Failed:
						status = fmt.Sprintf(Red, "[ FAILED ]   ")
						failedCount++
					case config.Disabled:
						status = fmt.Sprintf(Teal, "[ DISABLED ] ")
						failedCount++
					default:
						pendingCount++
					}

					res := fmt.Sprintf("%s.%s", r.Info().Type, r.Info().Name)
					fqdn := utils.FQDN(r.Info().Name, string(r.Info().Type))

					switch r.Info().Type {
					case config.TypeNomadCluster:
						fmt.Printf("%-13s %-30s %s\n", status, res, fmt.Sprintf("%s.%s", "server", utils.FQDN(r.Info().Name, string(r.Info().Type))))

						// add the client nodes
						nomad := r.(*config.NomadCluster)
						for n := 0; n < nomad.ClientNodes; n++ {
							fmt.Printf("%-13s %-30s %s\n", "", "", fmt.Sprintf("%d.%s.%s", n+1, "client", utils.FQDN(r.Info().Name, string(r.Info().Type))))
						}
					case config.TypeK8sCluster:
						fmt.Printf("%-13s %-30s %s\n", status, res, fmt.Sprintf("%s.%s", "server", utils.FQDN(r.Info().Name, string(r.Info().Type))))
					case config.TypeContainer:
						fallthrough
					case config.TypeSidecar:
						fallthrough
					case config.TypeK8sIngress:
						fallthrough
					case config.TypeNomadIngress:
						fallthrough
					case config.TypeContainerIngress:
						fallthrough
					case config.TypeImageCache:
						fmt.Printf("%-13s %-30s %s\n", status, res, fqdn)
					default:
						fmt.Printf("%-13s %-30s %s\n", status, res, "")
					}
				}
			}

			fmt.Println()
			fmt.Printf("Pending: %d Created: %d Failed: %d\n", pendingCount, createdCount, failedCount)
		}
	},
}

func init() {
	statusCmd.Flags().BoolVarP(&jsonFlag, "json", "", false, "Output the status as JSON")
	statusCmd.Flags().StringVarP(&resourceType, "type", "", "", "Resource type used to filter status list")
}
