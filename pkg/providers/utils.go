package providers

import (
	"fmt"
	"os"
	"strings"
)

// CreateKubeConfigPath creates the file path for the KubeConfig file when
// using a K3s or Kind cluster driver
func CreateKubeConfigPath(name string) (dir, filePath string, dockerPath string) {
	dir = fmt.Sprintf("%s/.shipyard/config/%s", os.Getenv("HOME"), name)
	filePath = fmt.Sprintf("%s/kubeconfig.yaml", dir)
	dockerPath = strings.Replace(filePath, ".yaml", "-docker.yaml", 1)

	return
}
