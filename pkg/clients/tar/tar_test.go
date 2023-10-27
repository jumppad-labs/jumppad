package tar

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func setupTarTests(t *testing.T) string {
	dir := t.TempDir()
	in := filepath.Join(dir, "in")
	out := filepath.Join(dir, "out")

	os.Mkdir(in, 0755)
	os.Mkdir(out, 0755)

	// write some files to the directory
	f, err := os.Create(filepath.Join(in, "test1.txt"))
	require.NoError(t, err)
	f.WriteString("test1")
	f.Close()

	f, err = os.Create(filepath.Join(in, "test2.txt"))
	require.NoError(t, err)
	f.WriteString("test2")
	f.Close()

	// create an empty directory
	os.Mkdir(filepath.Join(in, "/empty"), 0755)

	// create a sub directory with some files
	os.Mkdir(filepath.Join(in, "/sub"), 0755)

	f, err = os.Create(filepath.Join(in, "/sub", "test1.txt"))
	require.NoError(t, err)
	f.WriteString("test1")
	f.Close()

	f, err = os.Create(filepath.Join(in, "/sub", "test3.txt"))
	require.NoError(t, err)
	f.WriteString("test3")
	f.Close()

	return dir
}

func TestCompressedTarWithRootFolder(t *testing.T) {
	dir := setupTarTests(t)

	in := filepath.Join(dir, "in")
	out := filepath.Join(dir, "out")

	buf := bytes.NewBuffer(nil)

	tg := &TarGz{}

	// compress the directory
	err := tg.Create(buf, &TarGzOptions{ZipContents: true}, []string{in})
	require.NoError(t, err)

	os.WriteFile(filepath.Join(dir, "out.tar.gz"), buf.Bytes(), 0644)

	// test the output
	err = tg.Extract(buf, true, out)
	require.NoError(t, err)

	require.FileExists(t, filepath.Join(out, "/in/test1.txt"))
	require.FileExists(t, filepath.Join(out, "/in/test2.txt"))
	require.DirExists(t, filepath.Join(out, "/in/empty"))
	require.FileExists(t, filepath.Join(out, "/in/sub/test3.txt"))
}

func TestCompressedTarOmmitingRoot(t *testing.T) {
	dir := setupTarTests(t)

	in := filepath.Join(dir, "in")
	out := filepath.Join(dir, "out")

	buf := bytes.NewBuffer(nil)

	tg := &TarGz{}
	opts := TarGzOptions{OmitRoot: true, ZipContents: true}

	// compress the directory
	err := tg.Create(buf, &opts, []string{in})
	require.NoError(t, err)

	// test the output
	err = tg.Extract(buf, true, out)
	require.NoError(t, err)

	require.FileExists(t, filepath.Join(out, "/test1.txt"))
	require.FileExists(t, filepath.Join(out, "/test2.txt"))
	require.DirExists(t, filepath.Join(out, "/empty"))
	require.FileExists(t, filepath.Join(out, "/sub/test3.txt"))
}

func TestTarIndividualFiles(t *testing.T) {
	dir := setupTarTests(t)

	in := filepath.Join(dir, "in")
	out := filepath.Join(dir, "out")

	buf := bytes.NewBuffer(nil)

	tg := &TarGz{}
	opts := TarGzOptions{OmitRoot: true}

	err := tg.Create(buf, &opts, []string{filepath.Join(in, "test1.txt"), filepath.Join(in, "test2.txt")})
	require.NoError(t, err)

	// test the output
	err = tg.Extract(buf, false, out)
	require.NoError(t, err)

	require.FileExists(t, filepath.Join(out, "/test1.txt"))
	require.FileExists(t, filepath.Join(out, "/test2.txt"))
	require.NoFileExists(t, filepath.Join(out, "/sub/test3.txt"))
}

func TestTarDirAndIndividualFile(t *testing.T) {
	dir := setupTarTests(t)

	in := filepath.Join(dir, "in")
	out := filepath.Join(dir, "out")

	f, err := os.Create(filepath.Join(dir, "solo.txt"))
	require.NoError(t, err)
	f.WriteString("test3")
	f.Close()

	buf := bytes.NewBuffer(nil)

	tg := &TarGz{}
	opts := TarGzOptions{OmitRoot: true}

	err = tg.Create(buf, &opts, []string{in, filepath.Join(dir, "solo.txt")})
	require.NoError(t, err)

	// test the output
	err = tg.Extract(buf, false, out)
	require.NoError(t, err)

	require.FileExists(t, filepath.Join(out, "/test1.txt"))
	require.FileExists(t, filepath.Join(out, "/test2.txt"))
	require.FileExists(t, filepath.Join(out, "/solo.txt"))
	require.FileExists(t, filepath.Join(out, "/sub/test3.txt"))
}

func TestTarDirIgnoringFiles(t *testing.T) {
	dir := setupTarTests(t)

	in := filepath.Join(dir, "in")
	out := filepath.Join(dir, "out")

	buf := bytes.NewBuffer(nil)

	tg := &TarGz{}
	opts := TarGzOptions{OmitRoot: true}

	err := tg.Create(buf, &opts, []string{in}, "**/test1.txt", "**/sub")
	require.NoError(t, err)

	// test the output
	err = tg.Extract(buf, false, out)
	require.NoError(t, err)

	require.NoFileExists(t, filepath.Join(out, "/test1.txt"))
	require.FileExists(t, filepath.Join(out, "/test2.txt"))
	require.NoFileExists(t, filepath.Join(out, "/sub/test3.txt"))
}

func TestTarDirStrippingFolders(t *testing.T) {
	dir := setupTarTests(t)

	in := filepath.Join(dir, "in")
	out := filepath.Join(dir, "out")

	buf := bytes.NewBuffer(nil)

	tg := &TarGz{}
	opts := TarGzOptions{OmitRoot: false, StripFolders: true}

	err := tg.Create(buf, &opts, []string{in})
	require.NoError(t, err)

	// test the output
	err = tg.Extract(buf, false, out)
	require.NoError(t, err)

	require.FileExists(t, filepath.Join(out, "/test1.txt"))
	require.FileExists(t, filepath.Join(out, "/test2.txt"))
	require.FileExists(t, filepath.Join(out, "/test3.txt"))
}
