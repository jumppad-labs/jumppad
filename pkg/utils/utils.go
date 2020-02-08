package utils

import (
	"fmt"
	"os"
)

// FQDN generates the full qualified name for a container
func FQDN(name string, networkName string) string {
	if networkName == "" {
		return fmt.Sprintf("%s.shipyard", name)
	}

	return fmt.Sprintf("%s.%s.shipyard", name, networkName)
}

// CreateKubeConfigPath creates the file path for the KubeConfig file when
// using Kubernetes cluster
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

// FQDNVolumeName creates a full qualified volume name
func FQDNVolumeName(name string) string {
	return fmt.Sprintf("%s.volume.shipyard", name)
}
