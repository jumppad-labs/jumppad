package providers

import (
	"fmt"
	"os"
)

// CreateKubeConfigPath creates the file path for the KubeConfig file when
// using a K3s or Kind cluster driver
func CreateKubeConfigPath(name string) (dir, filePath string, dockerPath string) {
	dir = fmt.Sprintf("%s/.shipyard/config/%s", os.Getenv("HOME"), name)
	filePath = fmt.Sprintf("%s/kubeconfig.yaml", dir)
	dockerPath = fmt.Sprintf("%s/kubeconfig-docker.yaml", dir)

	// create the folders
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		panic(err)
	}

	return
}
