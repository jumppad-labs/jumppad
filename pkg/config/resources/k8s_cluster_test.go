package resources

import (
	"os"
	"testing"

	"github.com/shipyard-run/hclconfig/types"
	"github.com/stretchr/testify/require"
)

func TestK8sClusterProcessSetsAbsolute(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	c := &K8sCluster{
		ResourceMetadata: types.ResourceMetadata{File: "./"},
		Volumes: []Volume{
			{
				Source:      "./",
				Destination: "./",
			},
		},
	}

	c.Process()

	require.Equal(t, wd, c.Volumes[0].Source)
}

func TestK8sClusterSetsOutputsFromState(t *testing.T) {
	setupState(t, `
{
  "blueprint": null,
  "resources": [
	{
			"id": "resource.k8s_cluster.test",
      "name": "test",
      "status": "created",
      "type": "k8s_cluster",
			"address": "127.0.0.1",
			"api_port": 123,
			"connector_port": 124,
			"kubeconfig": "./mine.yaml"
	}
	]
}`)

	c := &K8sCluster{
		ResourceMetadata: types.ResourceMetadata{
			ID: "resource.k8s_cluster.test",
		},
	}

	c.Process()

	require.Equal(t, "127.0.0.1", c.Address)
	require.Equal(t, 123, c.APIPort)
	require.Equal(t, 124, c.ConnectorPort)
	require.Equal(t, "./mine.yaml", c.KubeConfig)
}
