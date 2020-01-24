package providers

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	hclog "github.com/hashicorp/go-hclog"
	clients "github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupRemoteExec(c *config.RemoteExec) (*clients.MockDocker, *RemoteExec, func()) {
	// set the shipyard env
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", "/tmp")

	md := &clients.MockDocker{}

	// Check that the image for the container exists
	md.On("ImageList", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	md.On("ImagePull", mock.Anything, mock.Anything, mock.Anything).Return(
		ioutil.NopCloser(strings.NewReader("")),
		nil,
	)

	// Create the container
	md.On("ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(container.ContainerCreateCreatedBody{}, nil)
	md.On("ContainerStart", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Remote the container
	md.On("ContainerRemove", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// lookup after container create and destroy
	md.On("ContainerList", mock.Anything, mock.Anything).Return([]types.Container{{ID: "tools"}}, nil)

	// container exec params
	md.On("ContainerExecCreate", mock.Anything, mock.Anything, mock.Anything).Return(types.IDResponse{ID: "abc"}, nil)
	md.On("ContainerExecAttach", mock.Anything, "abc", mock.Anything).Return(
		types.HijackedResponse{
			&net.TCPConn{},
			bufio.NewReader(bytes.NewReader([]byte("Do exec"))),
		},
		nil,
	)
	md.On("ContainerExecStart", mock.Anything, "abc", mock.Anything).Return(nil)
	md.On("ContainerExecInspect", mock.Anything, "abc").Return(nil, nil)

	return md, NewRemoteExec(c, md, hclog.Default()), func() {
		// cleanup
		os.Setenv("HOME", oldHome)
	}
}

func TestRemoteExecCreatesCorrectlyForContainer(t *testing.T) {
	c := &config.RemoteExec{
		TargetRef: config.Container{Name: "tester", NetworkRef: &config.Network{Name: "test"}},
		Command:   "/files/myscript.sh",
		Volumes: []config.Volume{
			config.Volume{
				Source:      "./files",
				Destination: "/files",
			},
		},
		Environment: []config.KV{
			config.KV{Key: "PATH", Value: "/usr/local/bin"},
		},
		WANRef: &config.Network{Name: "wan"},
	}

	md, re, cleanup := setupRemoteExec(c)
	defer cleanup()

	err := re.Create()
	assert.NoError(t, err)

	// check the ContainerCreate Parameters
	params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments
	dc := params[1].(*container.Config)
	hc := params[2].(*container.HostConfig)
	nc := params[3].(*network.NetworkingConfig)
	//fqdn := params[4]
	//cc :=

	// check the container is started with the correct image and command
	assert.Equal(t, dc.Image, c.Image.Name)
	assert.Equal(t, dc.Cmd[0], "tail")
	assert.Equal(t, dc.Cmd[1], "-f")
	assert.Equal(t, dc.Cmd[2], "/dev/null")

	// check that the volumes are correctly mounted
	assert.Equal(t, c.Volumes[0].Source, hc.Mounts[0].Source)
	assert.Equal(t, c.Volumes[0].Destination, hc.Mounts[0].Target)

	// ensure it connects to the wan network
	assert.NotNil(t, nc.EndpointsConfig[c.WANRef.Name])
	assert.Equal(t, c.WANRef.Name, nc.EndpointsConfig[c.WANRef.Name].NetworkID)

	// test environment variables are set
	assert.Equal(t, fmt.Sprintf("%s=%s", c.Environment[0].Key, c.Environment[0].Value), dc.Env[0])

	// check the details for the exec
	params = getCalls(&md.Mock, "ContainerExecCreate")[0].Arguments
	ec := params[2].(types.ExecConfig)

	assert.Equal(t, c.Command, ec.Cmd[0])
}
