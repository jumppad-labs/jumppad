package providers

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/mocks"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/stretchr/testify/mock"
	assert "github.com/stretchr/testify/require"
)

func TestContainerCreatesSuccessfully(t *testing.T) {
	cc := &resources.Container{ResourceMetadata: types.ResourceMetadata{
		Name: "tests",
	}}

	cc.Image = &resources.Image{}
	md := &clients.MockContainerTasks{}
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
	md := &clients.MockContainerTasks{}
	hc := &mocks.MockHTTP{}

	cc := &resources.Container{ResourceMetadata: types.ResourceMetadata{
		Name: "tests",
	}}

	cc.Depends = []string{"resource.network.test"}
	cc.Image = &resources.Image{Name: "abc"}
	cc.Volumes = []resources.Volume{resources.Volume{}}
	cc.Command = []string{"hello"}
	cc.Entrypoint = []string{"hello"}
	cc.Env = map[string]string{"hello": "world"}
	cc.HealthCheck = &resources.HealthCheck{}
	cc.Privileged = true
	cc.Resources = &resources.Resources{}
	cc.MaxRestartCount = 10

	md.On("PullImage", cc.Image, false).Once().Return(nil)
	md.On("CreateContainer", mock.Anything).Once().Return("", nil)

	c := NewContainerSidecar(cc, md, hc, hclog.NewNullLogger())
	err := c.Create()
	assert.NoError(t, err)

	ac := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*resources.Container)

	assert.Equal(t, cc.Name, ac.Name)
	assert.Equal(t, cc.Depends, ac.Depends)
	assert.Equal(t, cc.Volumes, ac.Volumes)
	assert.Equal(t, cc.Command, ac.Command)
	assert.Equal(t, cc.Entrypoint, ac.Entrypoint)
	assert.Equal(t, cc.Env, ac.Env)
	assert.Equal(t, cc.HealthCheck, ac.HealthCheck)
	assert.Equal(t, cc.Image.Name, ac.Image.Name)
	assert.Equal(t, cc.Privileged, ac.Privileged)
	assert.Equal(t, cc.Resources, ac.Resources)
	assert.Equal(t, cc.Type, ac.Type)
	assert.Equal(t, cc.MaxRestartCount, ac.MaxRestartCount)
}

func TestContainerRunsHTTPChecks(t *testing.T) {
	cc := &resources.Container{ResourceMetadata: types.ResourceMetadata{
		Name: "tests",
	}}
	cc.Image = &resources.Image{}
	cc.HealthCheck = &resources.HealthCheck{
		Timeout: "30s",
		HTTP:    "http://localhost:8500",
	}

	md := &clients.MockContainerTasks{}
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
	cc := &resources.Container{ResourceMetadata: types.ResourceMetadata{
		Name: "tests",
	}}
	cc.Image = &resources.Image{}
	cc.HealthCheck = &resources.HealthCheck{
		Timeout:          "30s",
		HTTP:             "http://localhost:8500",
		HTTPSuccessCodes: []int{200, 429},
	}

	md := &clients.MockContainerTasks{}
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
	cc := &resources.Container{ResourceMetadata: types.ResourceMetadata{
		Name: "tests",
	}}
	cc.Image = &resources.Image{}
	md := &clients.MockContainerTasks{}
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
	cc := &resources.Container{ResourceMetadata: types.ResourceMetadata{
		Name: "tests",
	}}
	cc.Networks = []resources.NetworkAttachment{resources.NetworkAttachment{Name: "cloud"}}
	md := &clients.MockContainerTasks{}
	hc := &mocks.MockHTTP{}
	c := NewContainer(cc, md, hc, hclog.NewNullLogger())

	md.On("FindContainerIDs", cc.Name, cc.Type).Return([]string{"abc"}, nil)
	md.On("RemoveContainer", "abc", false).Return(nil)
	md.On("DetachNetwork", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := c.Destroy()
	assert.NoError(t, err)
}

func TestContainerDoesNotDestroysWhenNotExists(t *testing.T) {
	cc := &resources.Container{ResourceMetadata: types.ResourceMetadata{
		Name: "tests",
	}}
	cc.Networks = []resources.NetworkAttachment{resources.NetworkAttachment{Name: "cloud"}}
	md := &clients.MockContainerTasks{}
	hc := &mocks.MockHTTP{}
	c := NewContainer(cc, md, hc, hclog.NewNullLogger())

	md.On("FindContainerIDs", cc.Name, cc.Type).Return(nil, nil)

	err := c.Destroy()
	assert.NoError(t, err)
	md.AssertNotCalled(t, "RemoveContainer")
}

func TestContainerDoesNotDestroysWhenLookupError(t *testing.T) {
	cc := &resources.Container{ResourceMetadata: types.ResourceMetadata{
		Name: "tests",
	}}
	cc.Networks = []resources.NetworkAttachment{resources.NetworkAttachment{Name: "cloud"}}
	md := &clients.MockContainerTasks{}
	hc := &mocks.MockHTTP{}
	c := NewContainer(cc, md, hc, hclog.NewNullLogger())

	md.On("FindContainerIDs", cc.Name, cc.Type).Return(nil, fmt.Errorf("boom"))

	err := c.Destroy()
	assert.Error(t, err)
	md.AssertNotCalled(t, "RemoveContainer")
}

func TestContainerLooksupIDs(t *testing.T) {
	cc := &resources.Container{ResourceMetadata: types.ResourceMetadata{
		Name: "tests",
	}}
	cc.Networks = []resources.NetworkAttachment{resources.NetworkAttachment{Name: "cloud"}}
	md := &clients.MockContainerTasks{}
	hc := &mocks.MockHTTP{}
	c := NewContainer(cc, md, hc, hclog.NewNullLogger())

	md.On("FindContainerIDs", cc.Name, cc.Type).Return([]string{"abc"}, nil)

	ids, err := c.Lookup()
	assert.NoError(t, err)
	assert.Equal(t, []string{"abc"}, ids)
}

func TestContainerBuildsContainer(t *testing.T) {
	cc := &resources.Container{ResourceMetadata: types.ResourceMetadata{
		Name: "tests",
	}}
	cc.Build = &resources.Build{Context: "./", File: "./"}

	md := &clients.MockContainerTasks{}
	md.On("BuildContainer", mock.Anything, mock.Anything).Return("testimage", nil)
	md.On("CreateContainer", cc).Once().Return("", nil)

	hc := &mocks.MockHTTP{}
	c := NewContainer(cc, md, hc, hclog.NewNullLogger())

	err := c.Create()
	assert.NoError(t, err)

	conf := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*resources.Container)
	assert.Equal(t, "testimage", conf.Image.Name)
}
