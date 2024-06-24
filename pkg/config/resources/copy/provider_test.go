package copy

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/getter"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/stretchr/testify/require"
)

func setupCopy(t *testing.T) (*Copy, *Provider) {
	dir := t.TempDir()
	inDir := path.Join(dir, "in")
	outDir := path.Join(dir, "out")

	// add a child dir
	err := os.Mkdir(inDir, 0775)
	require.NoError(t, err)

	// add some example files
	os.WriteFile(path.Join(inDir, "file1.txt"), []byte("file1"), 0755)
	os.WriteFile(path.Join(inDir, "file2.txt"), []byte("file2"), 0755)

	cc := &Copy{ResourceBase: types.ResourceBase{Meta: types.Meta{ID: "tests"}}}
	cc.Source = inDir
	cc.Destination = outDir

	p := &Provider{logger.NewTestLogger(t), cc, getter.NewGetter(true)}

	return cc, p
}

func TestCopiesADirectory(t *testing.T) {
	c, p := setupCopy(t)

	err := p.Create(context.Background())
	require.NoError(t, err)

	// check the destination
	require.DirExists(t, c.Destination)

	// check the files
	require.FileExists(t, path.Join(c.Destination, "file1.txt"))
	require.FileExists(t, path.Join(c.Destination, "file2.txt"))
}

func TestCopiesASingleFile(t *testing.T) {
	c, p := setupCopy(t)

	dDir := c.Destination
	c.Source = path.Join(c.Source, "file1.txt")
	c.Destination = path.Join(c.Destination, "file1.txt")

	err := p.Create(context.Background())
	require.NoError(t, err)

	// check the files
	require.FileExists(t, path.Join(dDir, "file1.txt"))
	require.NoFileExists(t, path.Join(dDir, "file2.txt"))
}

func TestCopiesADirectoryWithPermissions(t *testing.T) {
	c, p := setupCopy(t)
	c.Permissions = "0777"

	err := p.Create(context.Background())
	require.NoError(t, err)

	// check the destination
	require.DirExists(t, c.Destination)

	// check the files
	require.FileExists(t, path.Join(c.Destination, "file1.txt"))
	require.FileExists(t, path.Join(c.Destination, "file2.txt"))

	fs, _ := os.Stat(path.Join(c.Destination, "file1.txt"))
	require.Equal(t, os.FileMode(0777), fs.Mode())
}

func TestRemovesFiles(t *testing.T) {
	c, p := setupCopy(t)

	err := p.Create(context.Background())
	require.NoError(t, err)

	err = p.Destroy(context.Background(), false)
	require.NoError(t, err)

	require.NoFileExists(t, path.Join(c.Destination, "file1.txt"))
	require.NoFileExists(t, path.Join(c.Destination, "file2.txt"))
}

func TestFetchesRemoteFiles(t *testing.T) {
	c, p := setupCopy(t)
	c.Source = "https://raw.githubusercontent.com/jumppad-labs/jumppad/main/README.md"

	err := p.Create(context.Background())
	require.NoError(t, err)

	err = p.Destroy(context.Background(), false)
	require.NoError(t, err)

	require.FileExists(t, path.Join(c.Destination, "README.md"))
}
