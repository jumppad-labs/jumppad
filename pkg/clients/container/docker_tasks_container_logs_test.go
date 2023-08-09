package container

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/container/mocks"
	imocks "github.com/jumppad-labs/jumppad/pkg/clients/images/mocks"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/clients/tar"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestContainerLogsCalled(t *testing.T) {
	md := &mocks.Docker{}
	md.On("ServerVersion", mock.Anything).Return(types.Version{}, nil)
	md.On("ContainerLogs", mock.Anything, mock.Anything, mock.Anything).Return(
		ioutil.NopCloser(bytes.NewBufferString("test")),
		fmt.Errorf("boom"),
	)

	md.On("Info", mock.Anything).Return(types.Info{Driver: StorageDriverOverlay2}, nil)

	mic := &imocks.ImageLog{}

	dt := NewDockerTasks(md, mic, &tar.TarGz{}, logger.NewTestLogger(t))

	rc, err := dt.ContainerLogs("123", true, true)
	assert.NotNil(t, rc)
	assert.Error(t, err)
}
