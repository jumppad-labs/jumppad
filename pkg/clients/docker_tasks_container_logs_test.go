package clients

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/mocks"
	clients "github.com/jumppad-labs/jumppad/pkg/clients/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestContainerLogsCalled(t *testing.T) {
	md := &mocks.MockDocker{}
	md.On("ServerVersion", mock.Anything).Return(types.Version{}, nil)
	md.On("ContainerLogs", mock.Anything, mock.Anything, mock.Anything).Return(
		ioutil.NopCloser(bytes.NewBufferString("test")),
		fmt.Errorf("boom"),
	)

	md.On("Info", mock.Anything).Return(types.Info{Driver: StorageDriverOverlay2}, nil)

	mic := &mocks.ImageLog{}

	dt := NewDockerTasks(md, mic, &TarGz{}, clients.NewTestLogger(t))

	rc, err := dt.ContainerLogs("123", true, true)
	assert.NotNil(t, rc)
	assert.Error(t, err)
}
