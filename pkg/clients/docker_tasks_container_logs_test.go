package clients

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestContainerLogsCalled(t *testing.T) {
	md := &mocks.MockDocker{}
	md.On("ContainerLogs", mock.Anything, mock.Anything, mock.Anything).Return(
		ioutil.NopCloser(bytes.NewBufferString("test")),
		fmt.Errorf("boom"),
	)
	mic := &mocks.ImageLog{}

	dt := NewDockerTasks(md, mic, &TarGz{}, hclog.NewNullLogger())

	rc, err := dt.ContainerLogs("123", true, true)
	assert.NotNil(t, rc)
	assert.Error(t, err)
}
