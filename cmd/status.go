package cmd

import (
	"fmt"
	"os"

	"github.com/hokaccha/go-prettyjson"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/cache"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/container"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/k8s"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/nomad"
	"github.com/jumppad-labs/jumppad/pkg/jumppad/constants"
	"github.com/jumppad-labs/jumppad/pkg/utils"
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

		cfg, err := config.LoadState()
		if err != nil {
			fmt.Println(err)
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
			// fmt.Println()
			// fmt.Printf("%-13s %-60s %s\n", "STATUS", "RESOURCE", "FQDN")

			createdCount := 0
			failedCount := 0
			disabledCount := 0
			pendingCount := 0

			// sort the resources
			resourceMap := map[string][]types.Resource{}

			for _, r := range cfg.Resources {
				if resourceMap[r.Metadata().ResourceType] == nil {
					resourceMap[r.Metadata().ResourceType] = []types.Resource{}
				}

				resourceMap[r.Metadata().ResourceType] = append(resourceMap[r.Metadata().ResourceType], r)
			}

			for _, ress := range resourceMap {
				for _, r := range ress {
					if (resourceType != "" && r.Metadata().ResourceType != resourceType) ||
						r.Metadata().ResourceType == types.TypeModule ||
						r.Metadata().ResourceType == types.TypeVariable ||
						r.Metadata().ResourceType == types.TypeOutput {
						continue
					}

					status := yellowIcon.Render("?")
					if r.Metadata().Disabled {
						fmt.Printf("%s %s\n", grayIcon.Render("-"), grayText.Render(r.Metadata().ResourceID))
						disabledCount++
						continue
					} else {
						switch r.Metadata().ResourceProperties[constants.PropertyStatus] {
						case constants.StatusCreated:
							status = greenIcon.Render("✔")
							createdCount++
						case constants.StatusFailed:
							status = redIcon.Render("✘")
							failedCount++
						default:
							pendingCount++
						}
					}

					switch r.Metadata().ResourceType {
					case nomad.TypeNomadCluster:
						fmt.Printf("%s %s\n", status, r.Metadata().ResourceID)
						fmt.Printf("    %s %s\n", grayText.Render("└─"), whiteText.Render(fmt.Sprintf("%s.%s", "server", utils.FQDN(r.Metadata().ResourceName, r.Metadata().ResourceModule, string(r.Metadata().ResourceType)))))

						// add the client nodes
						nomad := r.(*nomad.NomadCluster)
						for n := 0; n < nomad.ClientNodes; n++ {
							fmt.Printf("    %s %s\n", grayText.Render("└─"), whiteText.Render(fmt.Sprintf("%d.%s.%s", n+1, "client", utils.FQDN(r.Metadata().ResourceName, r.Metadata().ResourceModule, string(r.Metadata().ResourceType)))))
						}
					case k8s.TypeK8sCluster:
						fmt.Printf("%s %s\n", status, r.Metadata().ResourceID)
						fmt.Printf("    %s %s\n", grayText.Render("└─"), whiteText.Render(fmt.Sprintf("%s.%s", "server", utils.FQDN(r.Metadata().ResourceName, r.Metadata().ResourceModule, r.Metadata().ResourceType))))
					case container.TypeContainer:
						fmt.Printf("%s %s\n", status, r.Metadata().ResourceID)
						fmt.Printf("    %s %s\n", grayText.Render("└─"), whiteText.Render(fmt.Sprintf("%s", utils.FQDN(r.Metadata().ResourceName, r.Metadata().ResourceModule, string(r.Metadata().ResourceType)))))
					case container.TypeSidecar:
						fmt.Printf("%s %s\n", status, r.Metadata().ResourceID)
						fmt.Printf("    %s %s\n", grayText.Render("└─"), whiteText.Render(fmt.Sprintf("%s", utils.FQDN(r.Metadata().ResourceName, r.Metadata().ResourceModule, string(r.Metadata().ResourceType)))))
					case cache.TypeImageCache:
						fmt.Printf("%s %s\n", status, r.Metadata().ResourceID)
					default:
						fmt.Printf("%s %s\n", status, r.Metadata().ResourceID)
					}
				}
			}

			// fmt.Println(greenIcon.Render("✔") + whiteText.Render("resource.image_cache.default"))
			// fmt.Println(greenIcon.Render("✔") + whiteText.Render("resource.network.main"))
			// fmt.Println(greenIcon.Render("✔") + whiteText.Render("resource.container.api"))
			// fmt.Println(grayText.Render("   ├─ ") + whiteText.Render("api.container.jumppad.dev"))
			// fmt.Println(grayText.Render("   └─ ") + whiteText.Render("backend.container.jumppad.dev"))
			// fmt.Println(greenIcon.Render("✔") + whiteText.Render("resource.container.advertisements"))
			// fmt.Println(grayText.Render("   └─ ") + whiteText.Render("advertisements.container.jumppad.dev"))
			// fmt.Println(redIcon.Render("✘") + whiteText.Render("resource.container.payments"))
			// fmt.Println(yellowIcon.Render("?") + whiteText.Render("resource.container.database"))
			// fmt.Println()
			// fmt.Println(grayIcon.Render("-") + grayText.Render("resource.container.frontend"))
			fmt.Println()
			fmt.Println(whiteText.Render(fmt.Sprintf("Pending: %d  Created: %d  Failed: %d  Disabled: %d", pendingCount, createdCount, failedCount, disabledCount)))
			fmt.Println()
		}
	},
}

func init() {
	statusCmd.Flags().BoolVarP(&jsonFlag, "json", "", false, "Output the status as JSON")
	statusCmd.Flags().StringVarP(&resourceType, "type", "", "", "Resource type used to filter status list")
}
