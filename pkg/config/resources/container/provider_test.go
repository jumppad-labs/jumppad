package container

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/instruqt/jumppad/pkg/clients/container/mocks"
	ctypes "github.com/instruqt/jumppad/pkg/clients/container/types"
	hmocks "github.com/instruqt/jumppad/pkg/clients/http/mocks"
	"github.com/instruqt/jumppad/pkg/clients/logger"
	"github.com/instruqt/jumppad/pkg/config/resources/healthcheck"
	"github.com/instruqt/jumppad/testutils"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/stretchr/testify/mock"
	assert "github.com/stretchr/testify/require"
)

func setupContainerTests(t *testing.T) (*Container, *mocks.ContainerTasks, *hmocks.HTTP) {
	cc := &Container{ResourceBase: types.ResourceBase{
		Meta: types.Meta{Name: "tests", Type: TypeContainer},
	}}

	cc.Image = Image{Name: "consul"}

	md := &mocks.ContainerTasks{}
	hc := &hmocks.HTTP{}

	// check pulls image before creating container
	md.On("PullImage", ctypes.Image{Name: cc.Image.Name, Username: cc.Image.Username, Password: cc.Image.Password}, false).Once().Return(nil)

	// fetches the id of the pulled image, this is used to detect changes
	md.On("FindImageInLocalRegistry", ctypes.Image{Name: cc.Image.Name, Username: cc.Image.Username, Password: cc.Image.Password}).Once().Return("myimage", nil)

	// check calls CreateContainer with the config
	md.On("CreateContainer", mock.Anything).Once().Return("12345", nil)

	// after creation the
	md.On("ListNetworks", "12345").Once().Return(nil, nil)

	return cc, md, hc
}

func TestContainerCreatesSuccessfully(t *testing.T) {
	cc, md, hc := setupContainerTests(t)

	c := Provider{cc, nil, md, hc, logger.NewTestLogger(t)}

	err := c.Create(context.Background())
	assert.NoError(t, err)

	hc.AssertNotCalled(t, "HealthCheckHTTP", mock.Anything, mock.Anything)
}

func TestContainerSidecarCreatesContainerSuccessfully(t *testing.T) {
	c, md, hc := setupContainerTests(t)
	testutils.RemoveOn(&md.Mock, "CreateContainer")
	md.On("CreateContainer", mock.Anything).Once().Return("12345", nil)

	cs := &Sidecar{ResourceBase: types.ResourceBase{
		Meta: types.Meta{Name: "tests", Type: TypeSidecar},
	}}

	cs.Target = *c
	cs.Volumes = []Volume{Volume{}}
	cs.Command = []string{"hello"}
	cs.Entrypoint = []string{"hello"}
	cs.Environment = map[string]string{"hello": "world"}
	cs.HealthCheck = &healthcheck.HealthCheckContainer{}
	cs.Image = Image{Name: "consul"}
	cs.Privileged = true
	cs.Resources = &Resources{CPU: 1}
	cs.MaxRestartCount = 10

	co := &Container{}
	co.ResourceBase = cs.ResourceBase
	co.ContainerName = cs.ContainerName

	co.Networks = []NetworkAttachment{{ID: "tests.container.local.jmpd.in"}}
	co.Volumes = cs.Volumes
	co.Command = cs.Command
	co.Entrypoint = cs.Entrypoint
	co.Labels = cs.Labels
	co.Environment = cs.Environment
	co.HealthCheck = cs.HealthCheck
	co.Image = cs.Image
	co.Privileged = cs.Privileged
	co.Resources = cs.Resources
	co.MaxRestartCount = cs.MaxRestartCount

	p := Provider{config: co, sidecar: cs, client: md, httpClient: hc, log: logger.NewTestLogger(t)}
	err := p.Create(context.Background())
	assert.NoError(t, err)

	ac := testutils.GetCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*ctypes.Container)

	assert.Equal(t, "tests.sidecar.local.jmpd.in", ac.Name)
	assert.Equal(t, "tests.container.local.jmpd.in", ac.Networks[0].ID)
	assert.Equal(t, cs.Command, ac.Command)
	assert.Equal(t, cs.Entrypoint, ac.Entrypoint)
	assert.Equal(t, cs.Environment, ac.Environment)
	assert.Equal(t, cs.Image.Name, ac.Image.Name)
	assert.Equal(t, cs.Privileged, ac.Privileged)
	assert.Equal(t, cs.Resources.CPU, ac.Resources.CPU)
	assert.Equal(t, cs.MaxRestartCount, ac.MaxRestartCount)
}

func TestContainerRunsHTTPChecks(t *testing.T) {
	cc, md, hc := setupContainerTests(t)
	cc.HealthCheck = &healthcheck.HealthCheckContainer{
		Timeout: "30s",
		HTTP: []healthcheck.HealthCheckHTTP{healthcheck.HealthCheckHTTP{
			Address:      "http://localhost:8500",
			SuccessCodes: []int{200, 429},
		}},
	}

	p := Provider{config: cc, client: md, httpClient: hc, log: logger.NewTestLogger(t)}

	hc.On("HealthCheckHTTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := p.Create(context.Background())
	assert.NoError(t, err)

	hc.AssertCalled(t, "HealthCheckHTTP", "http://localhost:8500", "", mock.Anything, mock.Anything, []int{200, 429}, 30*time.Second)
}

func TestContainerRunsTCPChecks(t *testing.T) {
	cc, md, hc := setupContainerTests(t)
	cc.HealthCheck = &healthcheck.HealthCheckContainer{
		Timeout: "30s",
		TCP: []healthcheck.HealthCheckTCP{healthcheck.HealthCheckTCP{
			Address: "http://localhost:8500",
		}},
	}

	p := Provider{config: cc, client: md, httpClient: hc, log: logger.NewTestLogger(t)}

	hc.On("HealthCheckTCP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := p.Create(context.Background())
	assert.NoError(t, err)

	hc.AssertCalled(t, "HealthCheckTCP", "http://localhost:8500", 30*time.Second)
}

func TestContainerRunsExecChecksWithCommand(t *testing.T) {
	command := []string{"terraform", "apply"}
	cc, md, hc := setupContainerTests(t)
	cc.HealthCheck = &healthcheck.HealthCheckContainer{
		Timeout: "30s",
		Exec: []healthcheck.HealthCheckExec{healthcheck.HealthCheckExec{
			Command: command,
		}},
	}

	md.On("ExecuteCommand", "12345", command, mock.Anything, "/tmp", "", "", 30, mock.Anything).Return(0, nil)

	p := Provider{config: cc, client: md, httpClient: hc, log: logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.NoError(t, err)

	md.AssertNumberOfCalls(t, "ExecuteCommand", 1)
}

func TestContainerRunsExecChecksWithScript(t *testing.T) {
	cc, md, hc := setupContainerTests(t)
	cc.HealthCheck = &healthcheck.HealthCheckContainer{
		Timeout: "30s",
		Exec: []healthcheck.HealthCheckExec{healthcheck.HealthCheckExec{
			Script: `#!/bin/bash
				curl http://something.com
			`,
		}},
	}

	md.On("CopyFileToContainer", "12345", mock.Anything, mock.Anything).Return(nil)
	md.On("ExecuteCommand", "12345", []string{"sh", "/tmp/script.sh"}, mock.Anything, "/tmp", "", "", 30, mock.Anything).Return(0, nil)

	p := Provider{config: cc, client: md, httpClient: hc, log: logger.NewTestLogger(t)}

	err := p.Create(context.Background())
	assert.NoError(t, err)

	md.AssertNumberOfCalls(t, "ExecuteCommand", 1)
}

func TestContainerDoesNOTCreateWhenPullImageFail(t *testing.T) {
	cc, md, hc := setupContainerTests(t)
	p := Provider{config: cc, client: md, httpClient: hc, log: logger.NewTestLogger(t)}

	// check pulls image before creating container and return an erro
	testutils.RemoveOn(&md.Mock, "PullImage")
	imageErr := fmt.Errorf("Unable to pull image")
	md.On("PullImage", ctypes.Image{Name: cc.Image.Name}, false).Once().Return(imageErr)

	// check does not call CreateContainer with the config
	md.On("CreateContainer", cc).Times(0)

	err := p.Create(context.Background())
	assert.Equal(t, imageErr, err)
}

func TestContainerDestroysCorrectlyWhenContainerExists(t *testing.T) {
	cc, md, hc := setupContainerTests(t)
	cc.Networks = []NetworkAttachment{NetworkAttachment{Name: "cloud"}}
	p := Provider{config: cc, client: md, httpClient: hc, log: logger.NewTestLogger(t)}

	md.On("FindContainerIDs", cc.ContainerName).Return([]string{"abc"}, nil)
	md.On("RemoveContainer", "abc", false).Return(nil)
	md.On("DetachNetwork", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := p.Destroy(context.Background(), false)
	assert.NoError(t, err)
}

func TestContainerDoesNotDestroysWhenNotExists(t *testing.T) {
	cc, md, hc := setupContainerTests(t)
	cc.Networks = []NetworkAttachment{NetworkAttachment{Name: "cloud"}}
	p := Provider{config: cc, client: md, httpClient: hc, log: logger.NewTestLogger(t)}

	md.On("FindContainerIDs", cc.ContainerName).Return(nil, nil)

	err := p.Destroy(context.Background(), false)
	assert.NoError(t, err)
	md.AssertNotCalled(t, "RemoveContainer")
}

func TestContainerDoesNotDestroysWhenLookupError(t *testing.T) {
	cc, md, hc := setupContainerTests(t)
	cc.Networks = []NetworkAttachment{NetworkAttachment{Name: "cloud"}}
	p := Provider{config: cc, client: md, httpClient: hc, log: logger.NewTestLogger(t)}

	md.On("FindContainerIDs", cc.ContainerName).Return(nil, fmt.Errorf("boom"))

	err := p.Destroy(context.Background(), false)
	assert.Error(t, err)
	md.AssertNotCalled(t, "RemoveContainer")
}

func TestContainerLooksupIDs(t *testing.T) {
	cc, md, hc := setupContainerTests(t)
	cc.Networks = []NetworkAttachment{NetworkAttachment{Name: "cloud"}}
	p := Provider{config: cc, client: md, httpClient: hc, log: logger.NewTestLogger(t)}

	md.On("FindContainerIDs", cc.ContainerName).Return([]string{"abc"}, nil)

	ids, err := p.Lookup()
	assert.NoError(t, err)
	assert.Equal(t, []string{"abc"}, ids)
}

func TestContainerAddsResources(t *testing.T) {
	cc, md, hc := setupContainerTests(t)
	cc.Networks = []NetworkAttachment{NetworkAttachment{Name: "cloud"}}
	cc.Resources = &Resources{
		CPU:    1,
		CPUPin: []int{1},
		Memory: 1,
		GPU: &GPU{
			Driver:    "nvidia",
			DeviceIDs: []string{"1"},
		},
	}

	p := Provider{config: cc, client: md, httpClient: hc, log: logger.NewTestLogger(t)}
	p.Create(context.Background())

	ac := testutils.GetCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*ctypes.Container)
	assert.Equal(t, 1, ac.Resources.CPU)
	assert.Equal(t, 1, ac.Resources.Memory)
	assert.Equal(t, []int{1}, ac.Resources.CPUPin)
	assert.Equal(t, "nvidia", ac.Resources.GPU.Driver)
	assert.Equal(t, []string{"1"}, ac.Resources.GPU.DeviceIDs)
}
