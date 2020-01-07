package providers

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/hashicorp/go-hclog"
	"github.com/docker/go-connections/nat"
	clients "github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupDocs(c *config.Docs) (*clients.MockDocker, *Docs) {
	md := &clients.MockDocker{}
	md.On("ImageList", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	md.On("ImagePull", mock.Anything, mock.Anything, mock.Anything).Return(
		ioutil.NopCloser(strings.NewReader("")),
		nil,
	)
	md.On("ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(container.ContainerCreateCreatedBody{}, nil)
	md.On("ContainerStart", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	return md, &Docs{c, md, hclog.Default()}
}

func TestCreatesDocumentationContainer(t *testing.T) {
	n := &config.Network{Name: "wan", Subnet: "10.1.1.0/24"}
	c := &config.Docs{
		Name:   "testdoc",
		Path:   "/folder/docs",
		Port:   8080,
		WANRef: n,
	}
	md, p := setupDocs(c)

	err := p.Create()

	assert.NoError(t, err)

	md.AssertCalled(t, "ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	md.AssertCalled(t, "ImagePull", mock.Anything, "docker.io/shipyardrun/docs:latest", mock.Anything)
	md.AssertCalled(t, "ContainerStart", mock.Anything, mock.Anything, mock.Anything)

	// second call is create
	params := getCalls(&md.Mock, "ContainerCreate")[0].Arguments
	name := params[4].(string)
	hc := params[2].(*container.HostConfig)
	dc := params[1].(*container.Config)
	//network := params[3].(*network.NetworkingConfig)

	assert.Equal(t, "testdoc.wan.shipyard", name)

	assert.Equal(t, c.Name, dc.Hostname)
	assert.Equal(t, "shipyardrun/docs:latest", dc.Image)

	// check the mounts
	// TODO fix these tests
	/*
	assert.Equal(t, c.Path+"/docs", hc.Mounts[0].Source)
	assert.Equal(t, "/shipyard/docs", hc.Mounts[0].Target)

	//assert.Equal(t, c.Path+"/static", hc.Mounts[1].Source)
	//assert.Equal(t, "/app/website/static", hc.Mounts[1].Target)

	assert.Equal(t, c.Path+"/siteConfig.js", hc.Mounts[1].Source)
	assert.Equal(t, "/shipyard/siteConfig.js", hc.Mounts[1].Target)

	assert.Equal(t, c.Path+"/sidebar.json", hc.Mounts[2].Source)
	assert.Equal(t, "/shipyard/sidebar.json", hc.Mounts[2].Target)
	*/

	// check the ports
	dockerPort, _ := nat.NewPort("tcp", "3000")

	assert.Len(t, dc.ExposedPorts, 1)
	assert.NotNil(t, dc.ExposedPorts[dockerPort])
	assert.NotNil(t, hc.PortBindings[dockerPort])
	assert.Equal(t, "8080", hc.PortBindings[dockerPort][0].HostPort)
}

func TestCreatesTerminalContainer(t*testing.T) {
	n := &config.Network{Name: "wan", Subnet: "10.1.1.0/24"}
	c := &config.Docs{
		Name:   "testdoc",
		Path:   "/folder/docs",
		Port:   8080,
		WANRef: n,
	}
	md, p := setupDocs(c)

	err := p.Create()

	assert.NoError(t, err)

	md.AssertCalled(t, "ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	md.AssertCalled(t, "ImagePull", mock.Anything, "docker.io/shipyardrun/docs:latest", mock.Anything)
	md.AssertCalled(t, "ContainerStart", mock.Anything, mock.Anything, mock.Anything)

	// second call is create
	params := getCalls(&md.Mock, "ContainerCreate")[1].Arguments
	name := params[4].(string)
	hc := params[2].(*container.HostConfig)
	dc := params[1].(*container.Config)

	assert.Equal(t, "terminal.wan.shipyard", name)

	// check the mounts for the docker dock
	assert.Equal(t, "/var/run/docker.sock", hc.Mounts[0].Source)
	assert.Equal(t, "/var/run/docker.sock", hc.Mounts[0].Target)

	// check the ports
	dockerPort, _ := nat.NewPort("tcp", "27950")

	assert.Len(t, dc.ExposedPorts, 1)
	assert.NotNil(t, dc.ExposedPorts[dockerPort])
	assert.NotNil(t, hc.PortBindings[dockerPort])
	assert.Equal(t, "27950", hc.PortBindings[dockerPort][0].HostPort)
}