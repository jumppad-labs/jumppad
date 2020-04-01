package clients

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNomadConfigLoadsCorrectly(t *testing.T) {
	fp, tmpDir, _ := setupNomadTests(t)
	defer os.RemoveAll(tmpDir)

	nc := &NomadConfig{}
	err := nc.Load(fp)
	assert.NoError(t, err)

	assert.Equal(t, "http://localhost:4646", nc.Location)
}

func TestNomadConfigLoadReturnsErrorWhenFileNotExist(t *testing.T) {
	nc := &NomadConfig{}
	err := nc.Load("file.json")
	assert.Error(t, err)
}

func TestNomadConfiSavesFile(t *testing.T) {
	fp, tmpDir, _ := setupNomadTests(t)
	defer os.RemoveAll(tmpDir)

	nc := &NomadConfig{Location: "http://nomad:4646"}
	err := nc.Save(fp)
	assert.NoError(t, err)

	// check the old file was deleted and the new file was written
	nc2 := &NomadConfig{}
	err = nc2.Load(fp)
	assert.NoError(t, err)

	assert.Equal(t, "http://nomad:4646", nc2.Location)
}
