package resources

import (
	"os"
	"path"
	"testing"

	"github.com/shipyard-run/hclconfig/types"
	"github.com/stretchr/testify/require"
)

func TestNomadClusterProcessSetsAbsolute(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	c := &NomadCluster{
		ResourceMetadata: types.ResourceMetadata{File: "./"},

		ServerConfig: "./server_config.hcl",
		ClientConfig: "./client_config.hcl",
		ConsulConfig: "./consul_config.hcl",

		Volumes: []Volume{
			{
				Source:      "./",
				Destination: "./",
			},
		},
	}

	c.Process()

	require.Equal(t, path.Join(wd, "server_config.hcl"), c.ServerConfig)
	require.Equal(t, path.Join(wd, "client_config.hcl"), c.ClientConfig)
	require.Equal(t, path.Join(wd, "consul_config.hcl"), c.ConsulConfig)
	require.Equal(t, wd, c.Volumes[0].Source)
}

func TestNomadClusterSetsOutputsFromState(t *testing.T) {
	setupState(t, `
{
  "blueprint": null,
  "resources": [
	{
			"id": "resource.nomad_cluster.test",
      "name": "test",
      "status": "created",
      "type": "nomad_cluster",
			"api_port": 123,
			"connector_port": 124,
			"server_address": "127.0.0.1",
			"client_address": ["127.0.0.2","127.0.0.3"],
			"config_dir": "abc/123"
	}
	]
}`)

	c := &NomadCluster{
		ResourceMetadata: types.ResourceMetadata{
			ID: "resource.nomad_cluster.test",
		},
	}

	c.Process()

	require.Equal(t, "127.0.0.1", c.ServerAddress)
	require.Equal(t, []string{"127.0.0.2", "127.0.0.3"}, c.ClientAddress)
	require.Equal(t, 123, c.APIPort)
	require.Equal(t, 124, c.ConnectorPort)
	require.Equal(t, "abc/123", c.ConfigDir)
}
