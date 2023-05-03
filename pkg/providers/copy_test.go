package providers

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/stretchr/testify/require"
)

func setupCopy(t *testing.T) (*config.Copy, *Copy) {
	dir := t.TempDir()
	inDir := path.Join(dir, "in")
	outDir := path.Join(dir, "out")

	// add a child dir
	err := os.Mkdir(inDir, 0775)
	require.NoError(t, err)

	// add some example files
	ioutil.WriteFile(path.Join(inDir, "file1.txt"), []byte("data"), 0755)
	ioutil.WriteFile(path.Join(inDir, "file2.txt"), []byte("data"), 0755)

	cc := config.NewCopy("tests")
	cc.Source = inDir
	cc.Destination = outDir

	p := NewCopy(cc, hclog.New(&hclog.LoggerOptions{Level: hclog.Debug}))

	return cc, p
}

func TestCopiesADirectory(t *testing.T) {
	c, p := setupCopy(t)

	err := p.Create()
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

	err := p.Create()
	require.NoError(t, err)

	// check the files
	require.FileExists(t, path.Join(dDir, "file1.txt"))
	require.NoFileExists(t, path.Join(dDir, "file2.txt"))
}

func TestCopiesADirectoryWithPermissions(t *testing.T) {
	c, p := setupCopy(t)
	c.Permissions = "0777"

	err := p.Create()
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

	err := p.Create()
	require.NoError(t, err)

	err = p.Destroy()
	require.NoError(t, err)

	require.NoFileExists(t, path.Join(c.Destination, "file1.txt"))
	require.NoFileExists(t, path.Join(c.Destination, "file2.txt"))
}
