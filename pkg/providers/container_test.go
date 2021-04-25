package providers

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/stretchr/testify/mock"
	assert "github.com/stretchr/testify/require"
)

func TestContainerCreatesSuccessfully(t *testing.T) {
	cc := config.NewContainer("tests")
	cc.Image = &config.Image{}
	md := &mocks.MockContainerTasks{}
	hc := &mocks.MockHTTP{}
	c := NewContainer(cc, md, hc, hclog.NewNullLogger())

	// check pulls image before creating container
	md.On("PullImage", *cc.Image, false).Once().Return(nil)

	// check calls CreateContainer with the config
	md.On("CreateContainer", cc).Once().Return("", nil)

	err := c.Create()
	assert.NoError(t, err)

	hc.AssertNotCalled(t, "HealthCheckHTTP", mock.Anything, mock.Anything)
}

func TestContainerSidecarCreatesContainerSuccessfully(t *testing.T) {
	md := &mocks.MockContainerTasks{}
	hc := &mocks.MockHTTP{}

	cc := config.NewSidecar("test")
	cc.Depends = []string{"network.test"}
	cc.Image = config.Image{Name: "abc"}
	cc.Volumes = []config.Volume{config.Volume{}}
	cc.Command = []string{"hello"}
	cc.Entrypoint = []string{"hello"}
	cc.EnvVar = map[string]string{"hello": "world"}
	cc.HealthCheck = &config.HealthCheck{}
	cc.Privileged = true
	cc.Resources = &config.Resources{}
	cc.Config = &config.Config{}
	cc.MaxRestartCount = 10

	md.On("PullImage", cc.Image, false).Once().Return(nil)
	md.On("CreateContainer", mock.Anything).Once().Return("", nil)

	c := NewContainerSidecar(cc, md, hc, hclog.NewNullLogger())
	err := c.Create()
	assert.NoError(t, err)

	ac := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*config.Container)

	assert.Equal(t, cc.Name, ac.Name)
	assert.Equal(t, cc.Depends, ac.Depends)
	assert.Equal(t, cc.Volumes, ac.Volumes)
	assert.Equal(t, cc.Command, ac.Command)
	assert.Equal(t, cc.Entrypoint, ac.Entrypoint)
	assert.Equal(t, cc.Environment, ac.Environment)
	assert.Equal(t, cc.EnvVar, ac.EnvVar)
	assert.Equal(t, cc.HealthCheck, ac.HealthCheck)
	assert.Equal(t, cc.Image.Name, ac.Image.Name)
	assert.Equal(t, cc.Privileged, ac.Privileged)
	assert.Equal(t, cc.Resources, ac.Resources)
	assert.Equal(t, cc.Type, ac.Type)
	assert.Equal(t, cc.Config, ac.Config)
	assert.Equal(t, cc.MaxRestartCount, ac.MaxRestartCount)
}

func TestContainerRunsHTTPChecks(t *testing.T) {
	cc := config.NewContainer("tests")
	cc.Image = &config.Image{}
	cc.HealthCheck = &config.HealthCheck{
		Timeout: "30s",
		HTTP:    "http://localhost:8500",
	}

	md := &mocks.MockContainerTasks{}
	hc := &mocks.MockHTTP{}
	c := NewContainer(cc, md, hc, hclog.NewNullLogger())

	md.On("PullImage", *cc.Image, false).Once().Return(nil)
	md.On("CreateContainer", cc).Once().Return("", nil)

	hc.On("HealthCheckHTTP", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := c.Create()
	assert.NoError(t, err)

	hc.AssertCalled(t, "HealthCheckHTTP", "http://localhost:8500", []int{200}, 30*time.Second)
}

func TestContainerRunsHTTPChecksWithCustomStatusCodes(t *testing.T) {
	cc := config.NewContainer("tests")
	cc.Image = &config.Image{}
	cc.HealthCheck = &config.HealthCheck{
		Timeout:          "30s",
		HTTP:             "http://localhost:8500",
		HTTPSuccessCodes: []int{200, 429},
	}

	md := &mocks.MockContainerTasks{}
	hc := &mocks.MockHTTP{}
	c := NewContainer(cc, md, hc, hclog.NewNullLogger())

	md.On("PullImage", *cc.Image, false).Once().Return(nil)
	md.On("CreateContainer", cc).Once().Return("", nil)

	hc.On("HealthCheckHTTP", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := c.Create()
	assert.NoError(t, err)

	hc.AssertCalled(t, "HealthCheckHTTP", "http://localhost:8500", []int{200, 429}, 30*time.Second)
}

func TestContainerDoesNOTCreateWhenPullImageFail(t *testing.T) {
	cc := config.NewContainer("tests")
	cc.Image = &config.Image{}
	md := &mocks.MockContainerTasks{}
	hc := &mocks.MockHTTP{}
	c := NewContainer(cc, md, hc, hclog.NewNullLogger())

	// check pulls image before creating container and return an erro
	imageErr := fmt.Errorf("Unable to pull image")
	md.On("PullImage", *cc.Image, false).Once().Return(imageErr)

	// check does not call CreateContainer with the config
	md.On("CreateContainer", cc).Times(0)

	err := c.Create()
	assert.Equal(t, imageErr, err)
}

func TestContainerDestroysCorrectlyWhenContainerExists(t *testing.T) {
	cc := config.NewContainer("tests")
	cc.Networks = []config.NetworkAttachment{config.NetworkAttachment{Name: "cloud"}}
	md := &mocks.MockContainerTasks{}
	hc := &mocks.MockHTTP{}
	c := NewContainer(cc, md, hc, hclog.NewNullLogger())

	md.On("FindContainerIDs", cc.Name, cc.Type).Return([]string{"abc"}, nil)
	md.On("RemoveContainer", "abc").Return(nil)
	md.On("DetachNetwork", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := c.Destroy()
	assert.NoError(t, err)
}

func TestContainerDoesNotDestroysWhenNotExists(t *testing.T) {
	cc := config.NewContainer("tests")
	cc.Networks = []config.NetworkAttachment{config.NetworkAttachment{Name: "cloud"}}
	md := &mocks.MockContainerTasks{}
	hc := &mocks.MockHTTP{}
	c := NewContainer(cc, md, hc, hclog.NewNullLogger())

	md.On("FindContainerIDs", cc.Name, cc.Type).Return(nil, nil)

	err := c.Destroy()
	assert.NoError(t, err)
	md.AssertNotCalled(t, "RemoveContainer")
}

func TestContainerDoesNotDestroysWhenLookupError(t *testing.T) {
	cc := config.NewContainer("tests")
	cc.Networks = []config.NetworkAttachment{config.NetworkAttachment{Name: "cloud"}}
	md := &mocks.MockContainerTasks{}
	hc := &mocks.MockHTTP{}
	c := NewContainer(cc, md, hc, hclog.NewNullLogger())

	md.On("FindContainerIDs", cc.Name, cc.Type).Return(nil, fmt.Errorf("boom"))

	err := c.Destroy()
	assert.Error(t, err)
	md.AssertNotCalled(t, "RemoveContainer")
}

func TestContainerLooksupIDs(t *testing.T) {
	cc := config.NewContainer("tests")
	cc.Networks = []config.NetworkAttachment{config.NetworkAttachment{Name: "cloud"}}
	md := &mocks.MockContainerTasks{}
	hc := &mocks.MockHTTP{}
	c := NewContainer(cc, md, hc, hclog.NewNullLogger())

	md.On("FindContainerIDs", cc.Name, cc.Type).Return([]string{"abc"}, nil)

	ids, err := c.Lookup()
	assert.NoError(t, err)
	assert.Equal(t, []string{"abc"}, ids)
}

func TestContainerBuildsContainer(t *testing.T) {
	cc := config.NewContainer("tests")
	cc.Build = &config.Build{Context: "./", File: "./"}

	md := &mocks.MockContainerTasks{}
	md.On("BuildContainer", mock.Anything, mock.Anything).Return("testimage", nil)
	md.On("CreateContainer", cc).Once().Return("", nil)

	hc := &mocks.MockHTTP{}
	c := NewContainer(cc, md, hc, hclog.NewNullLogger())

	err := c.Create()
	assert.NoError(t, err)

	conf := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*config.Container)
	assert.Equal(t, "testimage", conf.Image.Name)
}
