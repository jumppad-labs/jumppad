package getter

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func setupGetter(t *testing.T, force bool, err error) (string, Getter, *string, *string) {
	fp := filepath.Join(t.TempDir(), strconv.Itoa(time.Now().Nanosecond()))

	getSrc := ""
	getDst := ""

	g := &GetterImpl{
		force: force,
		get: func(uri, dst, pwd string) error {
			getSrc = uri
			getDst = dst

			return err
		},
	}

	return fp, g, &getSrc, &getDst
}

func TestGetsFolder(t *testing.T) {
	tmpDir, g, gs, gd := setupGetter(t, false, nil)
	defer os.RemoveAll(tmpDir)
	outDir := filepath.Join(tmpDir, "consul")
	url := "github.com/shipyard-run/blueprints//consul-nomad?ref=v0.0.1"

	err := g.Get(url, outDir)
	assert.NoError(t, err)

	assert.Equal(t, url, *gs)
	assert.Equal(t, outDir, *gd)
}

func TestNotGetFolderWhenExists(t *testing.T) {
	tmpDir, g, gs, gd := setupGetter(t, false, nil)
	defer os.RemoveAll(tmpDir)
	outDir := filepath.Join(tmpDir, "consul")
	url := "github.com/shipyard-run/blueprints//consul-nomad?ref=v0.0.1"

	os.MkdirAll(outDir, os.ModePerm)

	err := g.Get(url, outDir)
	assert.NoError(t, err)

	assert.Equal(t, *gs, "")
	assert.Equal(t, *gd, "")
}

func TestGetsFolderWhenExistsAndForceTrue(t *testing.T) {
	tmpDir, g, gs, gd := setupGetter(t, true, nil)
	defer os.RemoveAll(tmpDir)
	outDir := filepath.Join(tmpDir, "consul")
	url := "github.com/shipyard-run/blueprints//consul-nomad?ref=v0.0.1"

	os.MkdirAll(outDir, os.ModePerm)

	err := g.Get(url, outDir)
	assert.NoError(t, err)

	assert.Equal(t, url, *gs)
	assert.Equal(t, outDir, *gd)
}

func TestGetFunctional(t *testing.T) {
	g := NewGetter(true)
	url := "github.com/jetstack/cert-manager?ref=v1.2.0/deploy/charts//cert-manager"
	dir := t.TempDir()

	fmt.Println(dir)

	err := g.Get(url, dir)
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, "values.yaml"))
}
