package clients

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigLoadsCorrectly(t *testing.T) {
	fp, tmpDir, _ := setupNomadTests(t)
	defer os.RemoveAll(tmpDir)

	nc := &ClusterConfig{}
	err := nc.Load(fp, "local")
	assert.NoError(t, err)

	assert.Equal(t, "localhost", nc.LocalAddress)
	assert.Equal(t, "server.dev.nomad_cluster.shipyard.run", nc.RemoteAddress)
}

func TestNomadConfigLoadReturnsErrorWhenFileNotExist(t *testing.T) {
	nc := &ClusterConfig{}
	err := nc.Load("file.json", "local")
	assert.Error(t, err)
}

func TestConfiSavesFile(t *testing.T) {
	fp, tmpDir, _ := setupNomadTests(t)
	defer os.RemoveAll(tmpDir)

	nc := &ClusterConfig{
		LocalAddress:  "nomad",
		RemoteAddress: "nomad.remote",
		APIPort:       4646,
	}

	err := nc.Save(fp)
	assert.NoError(t, err)

	// check the old file was deleted and the new file was written
	nc2 := &ClusterConfig{}
	err = nc2.Load(fp, LocalContext)
	assert.NoError(t, err)

	assert.Equal(t, "nomad", nc2.LocalAddress)
	assert.Equal(t, "nomad.remote", nc2.RemoteAddress)
}

func TestConfigReturnsAPIFQDN(t *testing.T) {
	nc := ClusterConfig{LocalAddress: "localhost", APIPort: 4646, context: LocalContext}

	assert.Equal(t, "http://localhost:4646", nc.APIAddress())
}

func TestConfigReturnsLocalAPIFQDNSSL(t *testing.T) {
	nc := ClusterConfig{LocalAddress: "localhost", APIPort: 4646, SSL: true, context: LocalContext}

	assert.Equal(t, "https://localhost:4646", nc.APIAddress())
}

func TestConfigReturnsRemoteAPIFQDNSSL(t *testing.T) {
	nc := ClusterConfig{LocalAddress: "localhost", RemoteAddress: "nomad.remote", APIPort: 4646, RemoteAPIPort: 4646, SSL: true, context: RemoteContext}

	assert.Equal(t, "https://nomad.remote:4646", nc.APIAddress())
}

func TestConfigReturnsConnectorFQDN(t *testing.T) {
	nc := ClusterConfig{LocalAddress: "localhost", ConnectorPort: 4646, context: LocalContext}

	assert.Equal(t, "localhost:4646", nc.ConnectorAddress())
}
