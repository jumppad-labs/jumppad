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
	err := nc.Load(fp)
	assert.NoError(t, err)

	assert.Equal(t, "localhost", nc.Address)
}

func TestNomadConfigLoadReturnsErrorWhenFileNotExist(t *testing.T) {
	nc := &ClusterConfig{}
	err := nc.Load("file.json")
	assert.Error(t, err)
}

func TestConfiSavesFile(t *testing.T) {
	fp, tmpDir, _ := setupNomadTests(t)
	defer os.RemoveAll(tmpDir)

	nc := &ClusterConfig{
		Address: "nomad",
		APIPort: 4646,
	}

	err := nc.Save(fp)
	assert.NoError(t, err)

	// check the old file was deleted and the new file was written
	nc2 := &ClusterConfig{}
	err = nc2.Load(fp)
	assert.NoError(t, err)

	assert.Equal(t, "nomad", nc2.Address)
}

func TestConfigReturnsAPIFQDN(t *testing.T) {
	nc := ClusterConfig{Address: "localhost", APIPort: 4646}

	assert.Equal(t, "http://localhost:4646", nc.APIAddress())
}
func TestConfigReturnsAPIFQDNSSL(t *testing.T) {
	nc := ClusterConfig{Address: "localhost", APIPort: 4646, SSL: true}

	assert.Equal(t, "https://localhost:4646", nc.APIAddress())
}

func TestConfigReturnsConnectorFQDN(t *testing.T) {
	nc := ClusterConfig{Address: "localhost", ConnectorPort: 4646}

	assert.Equal(t, "localhost:4646", nc.ConnectorAddress())
}
