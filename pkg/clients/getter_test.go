package clients

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func setupGetter(t *testing.T, force bool, err error) (string, Getter, *string, *string) {
	fp := filepath.Join(os.TempDir(), strconv.Itoa(time.Now().Nanosecond()))

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
	t.Skip()
	tmpDir, g, gs, gd := setupGetter(t, false, nil)
	defer os.RemoveAll(tmpDir)
	outDir := filepath.Join(tmpDir, "consul")

	err := g.Get("github.com/shipyard-run/blueprints//consul-nomad", outDir)
	assert.NoError(t, err)

	assert.Equal(t, *gs, "github.com/shipyard-run/blueprints//consul-nomad")
	assert.Equal(t, *gd, outDir)
}

func TestDoesNotGetFolderWhenExists(t *testing.T) {
	t.Skip()
	tmpDir, g, gs, gd := setupGetter(t, false, nil)
	defer os.RemoveAll(tmpDir)
	outDir := filepath.Join(tmpDir, "consul")
	os.MkdirAll(outDir, os.ModePerm)

	err := g.Get("github.com/shipyard-run/blueprints//consul-nomad", outDir)
	assert.NoError(t, err)

	assert.Equal(t, *gs, "")
	assert.Equal(t, *gd, "")
}

func TestDoesGetsFolderWhenExistsAndForceTrue(t *testing.T) {
	t.Skip()
	tmpDir, g, gs, gd := setupGetter(t, true, nil)
	defer os.RemoveAll(tmpDir)
	outDir := filepath.Join(tmpDir, "consul")
	os.MkdirAll(outDir, os.ModePerm)

	err := g.Get("github.com/shipyard-run/blueprints//consul-nomad", outDir)
	assert.NoError(t, err)

	assert.Equal(t, *gs, "github.com/shipyard-run/blueprints//consul-nomad")
	assert.Equal(t, *gd, outDir)
}
