package resources

import (
	"os"
	"testing"

	"github.com/jumppad-labs/hclconfig/types"
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
			"external_ip": "127.0.0.1",
			"api_port": 123,
			"connector_port": 124,
			"kubeconfig": "./mine.yaml",
			"fqdn": "fqdn.mine.com",
			"networks": [{
				"assigned_address": "10.5.0.2",
				"name": "cloud"
			}]
	}]
}`)

	c := &K8sCluster{
		ResourceMetadata: types.ResourceMetadata{
			ID: "resource.k8s_cluster.test",
		},
		Networks: []NetworkAttachment{
			NetworkAttachment{},
		},
	}

	c.Process()

	// check the output parameters
	require.Equal(t, "127.0.0.1", c.ExternalIP)
	require.Equal(t, 123, c.APIPort)
	require.Equal(t, 124, c.ConnectorPort)
	require.Equal(t, "./mine.yaml", c.KubeConfig)
	require.Equal(t, "fqdn.mine.com", c.FQRN)

	// check the netwok
	require.Equal(t, "10.5.0.2", c.Networks[0].AssignedAddress)
	require.Equal(t, "cloud", c.Networks[0].Name)
}
