package utils

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

const configNomad = `
{
  "local_address": "localhost",
  "remote_address": "server.dev.nomad_cluster.shipyard.run",
  "api_port": 64124,
  "remote_api_port": 4646,
  "connector_port": 64648,
  "node_count": 1,
  "ssl": false
}
`

func setupNomadTests(t *testing.T) string {
	tmp := t.TempDir()
	configFile := filepath.Join(tmp, "config.json")
	ioutil.WriteFile(configFile, []byte(configNomad), os.ModePerm)

	return configFile
}

func TestConfigLoadsCorrectly(t *testing.T) {
	fp := setupNomadTests(t)

	nc := &ClusterConfig{}
	err := nc.Load(fp)
	assert.NoError(t, err)

	assert.Equal(t, "localhost", nc.LocalAddress)
	assert.Equal(t, "server.dev.nomad_cluster.shipyard.run", nc.RemoteAddress)
}

func TestNomadConfigLoadReturnsErrorWhenFileNotExist(t *testing.T) {
	nc := &ClusterConfig{}
	err := nc.Load("file.json")
	assert.Error(t, err)
}

func TestConfiSavesFile(t *testing.T) {
	fp := setupNomadTests(t)

	nc := &ClusterConfig{
		LocalAddress:  "nomad",
		RemoteAddress: "nomad.remote",
		APIPort:       4646,
	}

	err := nc.Save(fp)
	assert.NoError(t, err)

	// check the old file was deleted and the new file was written
	nc2 := &ClusterConfig{}
	err = nc2.Load(fp)
	assert.NoError(t, err)

	assert.Equal(t, "nomad", nc2.LocalAddress)
	assert.Equal(t, "nomad.remote", nc2.RemoteAddress)
}

func TestConfigReturnsAPIFQDN(t *testing.T) {
	nc := ClusterConfig{LocalAddress: "localhost", APIPort: 4646}

	assert.Equal(t, "http://localhost:4646", nc.APIAddress(LocalContext))
}

func TestConfigReturnsLocalAPIFQDNSSL(t *testing.T) {
	nc := ClusterConfig{LocalAddress: "localhost", APIPort: 4646, SSL: true}

	assert.Equal(t, "https://localhost:4646", nc.APIAddress(LocalContext))
}

func TestConfigReturnsRemoteAPIFQDNSSL(t *testing.T) {
	nc := ClusterConfig{LocalAddress: "localhost", RemoteAddress: "nomad.remote", APIPort: 4646, RemoteAPIPort: 4646, SSL: true}

	assert.Equal(t, "https://nomad.remote:4646", nc.APIAddress(RemoteContext))
}

func TestConfigReturnsConnectorFQDN(t *testing.T) {
	nc := ClusterConfig{LocalAddress: "localhost", ConnectorPort: 4646}

	assert.Equal(t, "localhost:4646", nc.ConnectorAddress(LocalContext))
}
