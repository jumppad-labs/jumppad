package container

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/system"
	"github.com/jumppad-labs/jumppad/pkg/clients/container/mocks"
	imocks "github.com/jumppad-labs/jumppad/pkg/clients/images/mocks"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/clients/tar"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCopyFromContainerCopiesFile(t *testing.T) {
	id := "abc"
	src := "/output/file.txt"

	md := &mocks.Docker{}
	md.On("ServerVersion", mock.Anything).Return(types.Version{}, nil)
	md.On("Info", mock.Anything).Return(system.Info{Driver: StorageDriverOverlay2}, nil)

	tmpDir := t.TempDir()
	tgz := &tar.TarGz{}

	os.Mkdir(filepath.Join(tmpDir, "input"), 0755)
	os.Mkdir(filepath.Join(tmpDir, "output"), 0755)

	// create the test tar
	os.WriteFile(filepath.Join(tmpDir, "input", "file.txt"), []byte("test content"), 0644)

	buf := bytes.NewBuffer(nil)
	err := tgz.Create(buf, &tar.TarGzOptions{OmitRoot: true}, []string{filepath.Join(tmpDir, "input", "file.txt")})
	require.NoError(t, err)

	mic := &imocks.ImageLog{}
	md.On("CopyFromContainer", mock.Anything, id, src).Return(
		io.NopCloser(bytes.NewBuffer(buf.Bytes())),
		container.PathStat{},
		nil,
	)

	dt, _ := NewDockerTasks(md, mic, &tar.TarGz{}, logger.NewTestLogger(t))

	err = dt.CopyFromContainer(id, src, filepath.Join(tmpDir, "output", "file.txt"))
	require.NoError(t, err)

	// check the file was written correctly
	d, err := ioutil.ReadFile(filepath.Join(tmpDir, "output", "file.txt"))
	require.NoError(t, err)
	require.Equal(t, "test content", string(d))
}

func TestCopyFromContainerReturnsErrorOnDockerError(t *testing.T) {
	id := "abc"
	src := "/output/file.hcl"

	md := &mocks.Docker{}
	md.On("ServerVersion", mock.Anything).Return(types.Version{}, nil)
	md.On("Info", mock.Anything).Return(system.Info{Driver: StorageDriverOverlay2}, nil)

	mic := &imocks.ImageLog{}
	md.On("CopyFromContainer", mock.Anything, id, src).Return(
		nil,
		container.PathStat{},
		fmt.Errorf("boom"),
	)
	dt, _ := NewDockerTasks(md, mic, &tar.TarGz{}, logger.NewTestLogger(t))

	err := dt.CopyFromContainer(id, src, "/new.hcl")
	assert.Error(t, err)
}
