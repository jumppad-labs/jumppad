package clients

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/docker/pkg/ioutils"
	"github.com/stretchr/testify/assert"
)

func setupNomadClient() {

}

func createTestFile(t *testing.T) (string, string) {
	tmpDir, err := ioutils.TempDir("", "")
	assert.NoError(t, err)

	fp := filepath.Join(tmpDir, "nomad.json")
	f, err := os.Create(fp)
	assert.NoError(t, err)

	_, err = f.WriteString(testNomadConfig)
	assert.NoError(t, err)

	return fp, tmpDir
}

func TestNomadConfigLoadsCorrectly(t *testing.T) {
	fp, tmpDir := createTestFile(t)
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
	fp, tmpDir := createTestFile(t)
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

var testNomadConfig = `
	{
		"location": "http://localhost:4646"
	}
`
