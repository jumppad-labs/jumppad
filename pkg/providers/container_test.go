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

func setupContainerTests(t *testing.T) (*resources.Container, *clients.MockContainerTasks, *mocks.MockHTTP) {
	cc := &resources.Container{ResourceMetadata: types.ResourceMetadata{
		Name: "tests",
	}}

	cc.Image = &resources.Image{Name: "consul"}

	md := &clients.MockContainerTasks{}
	hc := &mocks.MockHTTP{}

	// check pulls image before creating container
	md.On("PullImage", *cc.Image, false).Once().Return(nil)

	// check calls CreateContainer with the config
	md.On("CreateContainer", cc).Once().Return("12345", nil)

	// after creation the
	md.On("ListNetworks", "12345").Once().Return(nil, nil)

	return cc, md, hc
}

func TestContainerCreatesSuccessfully(t *testing.T) {
	cc, md, hc := setupContainerTests(t)

	c := NewContainer(cc, md, hc, hclog.NewNullLogger())

	err := c.Create()
	assert.NoError(t, err)

	hc.AssertNotCalled(t, "HealthCheckHTTP", mock.Anything, mock.Anything)
}

func TestContainerSidecarCreatesContainerSuccessfully(t *testing.T) {
	_, md, hc := setupContainerTests(t)
	removeOn(&md.Mock, "CreateContainer")
	md.On("CreateContainer", mock.Anything).Once().Return("12345", nil)

	cs := &resources.Sidecar{ResourceMetadata: types.ResourceMetadata{
		Name: "tests",
	}}

	cs.Target = "resources.container.consul"
	cs.Volumes = []resources.Volume{resources.Volume{}}
	cs.Command = []string{"hello"}
	cs.Entrypoint = []string{"hello"}
	cs.Environment = map[string]string{"hello": "world"}
	cs.HealthCheck = &resources.HealthCheckContainer{}
	cs.Image = resources.Image{Name: "consul"}
	cs.Privileged = true
	cs.Resources = &resources.Resources{}
	cs.MaxRestartCount = 10

	c := NewContainerSidecar(cs, md, hc, hclog.NewNullLogger())
	err := c.Create()
	assert.NoError(t, err)

	ac := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*resources.Container)

	assert.Equal(t, cs.Name, ac.Name)
	assert.Equal(t, "resources.container.consul", ac.Networks[0].ID)
	assert.Equal(t, cs.Volumes, ac.Volumes)
	assert.Equal(t, cs.Command, ac.Command)
	assert.Equal(t, cs.Entrypoint, ac.Entrypoint)
	assert.Equal(t, cs.Environment, ac.Environment)
	assert.Equal(t, cs.HealthCheck, ac.HealthCheck)
	assert.Equal(t, &cs.Image, ac.Image)
	assert.Equal(t, cs.Privileged, ac.Privileged)
	assert.Equal(t, cs.Resources, ac.Resources)
	assert.Equal(t, cs.MaxRestartCount, ac.MaxRestartCount)
}

func TestContainerRunsHTTPChecks(t *testing.T) {
	cc, md, hc := setupContainerTests(t)
	cc.HealthCheck = &resources.HealthCheckContainer{
		Timeout: "30s",
		HTTP: []resources.HealthCheckHTTP{resources.HealthCheckHTTP{
			Address:      "http://localhost:8500",
			SuccessCodes: []int{200, 429},
		}},
	}

	c := NewContainer(cc, md, hc, hclog.NewNullLogger())

	hc.On("HealthCheckHTTP", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := c.Create()
	assert.NoError(t, err)

	hc.AssertCalled(t, "HealthCheckHTTP", "http://localhost:8500", []int{200, 429}, 30*time.Second)
}

func TestContainerRunsTCPChecks(t *testing.T) {
	cc, md, hc := setupContainerTests(t)
	cc.HealthCheck = &resources.HealthCheckContainer{
		Timeout: "30s",
		TCP: []resources.HealthCheckTCP{resources.HealthCheckTCP{
			Address: "http://localhost:8500",
		}},
	}

	c := NewContainer(cc, md, hc, hclog.NewNullLogger())

	hc.On("HealthCheckTCP", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := c.Create()
	assert.NoError(t, err)

	hc.AssertCalled(t, "HealthCheckTCP", "http://localhost:8500", 30*time.Second)
}

func TestContainerRunsExecChecksWithCommand(t *testing.T) {
	command := []string{"terraform", "apply"}
	cc, md, hc := setupContainerTests(t)
	cc.HealthCheck = &resources.HealthCheckContainer{
		Timeout: "30s",
		Exec: []resources.HealthCheckExec{resources.HealthCheckExec{
			Command: command,
		}},
	}

	md.On("ExecuteCommand", "12345", command, mock.Anything, "/tmp", "", "", mock.Anything).Return(0, nil)

	c := NewContainer(cc, md, hc, hclog.NewNullLogger())

	err := c.Create()
	assert.NoError(t, err)

	md.AssertNumberOfCalls(t, "ExecuteCommand", 1)
}

func TestContainerRunsExecChecksWithScript(t *testing.T) {
	cc, md, hc := setupContainerTests(t)
	cc.HealthCheck = &resources.HealthCheckContainer{
		Timeout: "30s",
		Exec: []resources.HealthCheckExec{resources.HealthCheckExec{
			Script: `#!/bin/bash
				curl http://something.com
			`,
		}},
	}

	md.On("CopyFileToContainer", "12345", mock.Anything, mock.Anything).Return(nil)
	md.On("ExecuteCommand", "12345", []string{"sh", "/tmp/script.sh"}, mock.Anything, "/tmp", "", "", mock.Anything).Return(0, nil)

	c := NewContainer(cc, md, hc, hclog.NewNullLogger())

	err := c.Create()
	assert.NoError(t, err)

	md.AssertNumberOfCalls(t, "ExecuteCommand", 1)
}

func TestContainerDoesNOTCreateWhenPullImageFail(t *testing.T) {
	cc, md, hc := setupContainerTests(t)
	c := NewContainer(cc, md, hc, hclog.NewNullLogger())

	// check pulls image before creating container and return an erro
	removeOn(&md.Mock, "PullImage")
	imageErr := fmt.Errorf("Unable to pull image")
	md.On("PullImage", *cc.Image, false).Once().Return(imageErr)

	// check does not call CreateContainer with the config
	md.On("CreateContainer", cc).Times(0)

	err := c.Create()
	assert.Equal(t, imageErr, err)
}

func TestContainerDestroysCorrectlyWhenContainerExists(t *testing.T) {
	cc, md, hc := setupContainerTests(t)
	cc.Networks = []resources.NetworkAttachment{resources.NetworkAttachment{Name: "cloud"}}
	c := NewContainer(cc, md, hc, hclog.NewNullLogger())

	md.On("FindContainerIDs", cc.FQRN).Return([]string{"abc"}, nil)
	md.On("RemoveContainer", "abc", false).Return(nil)
	md.On("DetachNetwork", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := c.Destroy()
	assert.NoError(t, err)
}

func TestContainerDoesNotDestroysWhenNotExists(t *testing.T) {
	cc, md, hc := setupContainerTests(t)
	cc.Networks = []resources.NetworkAttachment{resources.NetworkAttachment{Name: "cloud"}}
	c := NewContainer(cc, md, hc, hclog.NewNullLogger())

	md.On("FindContainerIDs", cc.FQRN).Return(nil, nil)

	err := c.Destroy()
	assert.NoError(t, err)
	md.AssertNotCalled(t, "RemoveContainer")
}

func TestContainerDoesNotDestroysWhenLookupError(t *testing.T) {
	cc, md, hc := setupContainerTests(t)
	cc.Networks = []resources.NetworkAttachment{resources.NetworkAttachment{Name: "cloud"}}
	c := NewContainer(cc, md, hc, hclog.NewNullLogger())

	md.On("FindContainerIDs", cc.FQRN).Return(nil, fmt.Errorf("boom"))

	err := c.Destroy()
	assert.Error(t, err)
	md.AssertNotCalled(t, "RemoveContainer")
}

func TestContainerLooksupIDs(t *testing.T) {
	cc, md, hc := setupContainerTests(t)
	cc.Networks = []resources.NetworkAttachment{resources.NetworkAttachment{Name: "cloud"}}
	c := NewContainer(cc, md, hc, hclog.NewNullLogger())

	md.On("FindContainerIDs", cc.FQRN).Return([]string{"abc"}, nil)

	ids, err := c.Lookup()
	assert.NoError(t, err)
	assert.Equal(t, []string{"abc"}, ids)
}

func TestContainerBuildsContainer(t *testing.T) {
	cc, md, hc := setupContainerTests(t)
	cc.Build = &resources.Build{Context: "./"}

	md.On("BuildContainer", mock.Anything, mock.Anything).Return("testimage", nil)

	c := NewContainer(cc, md, hc, hclog.NewNullLogger())

	err := c.Create()
	assert.NoError(t, err)

	conf := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*resources.Container)
	assert.Equal(t, "testimage", conf.Image.Name)
}