package providers

import (
	"fmt"
	"os"
	"strings"

	"github.com/shipyard-run/cli/pkg/config"
)

// FQDN generate the name of a docker container
func FQDN(name string, net *config.Network) string {
	if net == nil {
		return fmt.Sprintf("%s.shipyard", name)
	}

	return fmt.Sprintf("%s.%s.shipyard", name, net.Name)
}

// CreateKubeConfigPath creates the file path for the KubeConfig file when
// using a K3s or Kind cluster driver
func CreateKubeConfigPath(name string) (dir, filePath string, dockerPath string) {
	dir = fmt.Sprintf("%s/.shipyard/config/%s", os.Getenv("HOME"), name)
	filePath = fmt.Sprintf("%s/kubeconfig.yaml", dir)
	dockerPath = strings.Replace(filePath, ".yaml", "-docker.yaml", 1)

	return
}
