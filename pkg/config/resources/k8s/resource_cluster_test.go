package k8s

import (
	"os"
	"testing"

	"github.com/instruqt/jumppad/pkg/config"
	ctypes "github.com/instruqt/jumppad/pkg/config/resources/container"
	"github.com/instruqt/jumppad/testutils"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/stretchr/testify/require"
)

func init() {
	config.RegisterResource(TypeK8sCluster, &Cluster{}, &ClusterProvider{})
}

func TestK8sClusterProcessSetsAbsolute(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	c := &Cluster{
		ResourceBase: types.ResourceBase{Meta: types.Meta{File: "./"}},
		Volumes: []ctypes.Volume{
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
	testutils.SetupState(t, `
{
  "blueprint": null,
  "resources": [
  {
      "meta": {
      "id": "resource.k8s_cluster.test",
      "name": "test",
      "type": "k8s_cluster"
      },
      "external_ip": "127.0.0.1",
      "api_port": 123,
      "connector_port": 124,
      "kube_config": {
				"path": "./mine.yaml"
			},
      "container_name": "fqdn.mine.com",
      "networks": [{
        "assigned_address": "10.5.0.2",
        "name": "cloud"
      }]
  }]
}`)

	c := &Cluster{
		ResourceBase: types.ResourceBase{
			Meta: types.Meta{
				ID: "resource.k8s_cluster.test",
			},
		},
		Networks: []ctypes.NetworkAttachment{
			ctypes.NetworkAttachment{},
		},
	}

	c.Process()

	// check the output parameters
	require.Equal(t, "127.0.0.1", c.ExternalIP)
	require.Equal(t, 123, c.APIPort)
	require.Equal(t, 124, c.ConnectorPort)
	require.Equal(t, "./mine.yaml", c.KubeConfig.ConfigPath)
	require.Equal(t, "fqdn.mine.com", c.ContainerName)

	// check the netwok
	require.Equal(t, "10.5.0.2", c.Networks[0].AssignedAddress)
	require.Equal(t, "cloud", c.Networks[0].Name)
}
