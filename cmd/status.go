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
					if (resourceType != "" && r.Metadata().Type != resourceType) ||
						r.Metadata().Type == types.TypeModule ||
						r.Metadata().Type == types.TypeVariable ||
						r.Metadata().Type == types.TypeOutput {
						continue
					}

					status := YellowIcon.Render("?")
					if r.GetDisabled() {
						fmt.Printf("%s %s\n", GrayIcon.Render("-"), GrayText.Render(r.Metadata().ID))
						disabledCount++
						continue
					} else {
						switch r.Metadata().Properties[constants.PropertyStatus] {
						case constants.StatusCreated:
							status = GreenIcon.Render("✔")
							createdCount++
						case constants.StatusFailed:
							status = RedIcon.Render("✘")
							failedCount++
						default:
							pendingCount++
						}
					}

					switch r.Metadata().Type {
					case nomad.TypeNomadCluster:
						fmt.Printf("%s %s\n", status, r.Metadata().ID)
						fmt.Printf("    %s %s\n", GrayText.Render("└─"), WhiteText.Render(fmt.Sprintf("%s.%s", "server", utils.FQDN(r.Metadata().Name, r.Metadata().Module, string(r.Metadata().Type)))))

						// add the client nodes
						nomad := r.(*nomad.NomadCluster)
						for n := 0; n < nomad.ClientNodes; n++ {
							fmt.Printf("    %s %s\n", GrayText.Render("└─"), WhiteText.Render(fmt.Sprintf("%d.%s.%s", n+1, "client", utils.FQDN(r.Metadata().Name, r.Metadata().Module, string(r.Metadata().Type)))))
						}
					case k8s.TypeK8sCluster:
						fmt.Printf("%s %s\n", status, r.Metadata().ID)
						fmt.Printf("    %s %s\n", GrayText.Render("└─"), WhiteText.Render(fmt.Sprintf("%s.%s", "server", utils.FQDN(r.Metadata().Name, r.Metadata().Module, r.Metadata().Type))))
					case container.TypeContainer:
						fmt.Printf("%s %s\n", status, r.Metadata().ID)
						fmt.Printf("    %s %s\n", GrayText.Render("└─"), WhiteText.Render(utils.FQDN(r.Metadata().Name, r.Metadata().Module, string(r.Metadata().Type))))
					case container.TypeSidecar:
						fmt.Printf("%s %s\n", status, r.Metadata().ID)
						fmt.Printf("    %s %s\n", GrayText.Render("└─"), WhiteText.Render(utils.FQDN(r.Metadata().Name, r.Metadata().Module, string(r.Metadata().Type))))
					case cache.TypeImageCache:
						fmt.Printf("%s %s\n", status, r.Metadata().ID)
					default:
						fmt.Printf("%s %s\n", status, r.Metadata().ID)
					}
				}
			}

			fmt.Println()
			fmt.Println(WhiteText.Render(fmt.Sprintf("Pending: %d  Created: %d  Failed: %d  Disabled: %d", pendingCount, createdCount, failedCount, disabledCount)))
			fmt.Println()
		}
	},
}

func init() {
	statusCmd.Flags().BoolVarP(&jsonFlag, "json", "", false, "Output the status as JSON")
	statusCmd.Flags().StringVarP(&resourceType, "type", "", "", "Resource type used to filter status list")
}
