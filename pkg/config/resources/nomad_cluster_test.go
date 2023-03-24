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
