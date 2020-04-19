package clients

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func setupGetter(t *testing.T) string {
	fp := filepath.Join(os.TempDir(), strconv.Itoa(time.Now().Nanosecond()))

	return fp
}

func TestGetsFolder(t *testing.T) {
	tmpDir := setupGetter(t)
	defer os.RemoveAll(tmpDir)
	outDir := filepath.Join(tmpDir, "consul")

	g := GetterImpl{}
	err := g.Get("github.com/shipyard-run/blueprints//consul-nomad", outDir)
	assert.NoError(t, err)

	assert.DirExists(t, outDir)
	assert.FileExists(t, filepath.Join(outDir, "README.md"))
}

func TestDoesNotGetFolderWhenExists(t *testing.T) {
	tmpDir := setupGetter(t)
	defer os.RemoveAll(tmpDir)
	outDir := filepath.Join(tmpDir, "consul")
	os.MkdirAll(outDir, os.ModePerm)

	g := GetterImpl{}
	err := g.Get("github.com/shipyard-run/blueprints//consul-nomad", outDir)
	assert.NoError(t, err)

	assert.DirExists(t, outDir)
	assert.NoFileExists(t, filepath.Join(outDir, "README.md"))
}

func TestDoesGetsFolderWhenExistsAndForceTrue(t *testing.T) {
	tmpDir := setupGetter(t)
	defer os.RemoveAll(tmpDir)
	outDir := filepath.Join(tmpDir, "consul")
	os.MkdirAll(outDir, os.ModePerm)

	g := GetterImpl{force: true}
	err := g.Get("github.com/shipyard-run/blueprints//consul-nomad", outDir)
	assert.NoError(t, err)

	assert.DirExists(t, outDir)
	assert.FileExists(t, filepath.Join(outDir, "README.md"))
}
