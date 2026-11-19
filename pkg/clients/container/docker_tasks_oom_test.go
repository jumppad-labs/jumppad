package container

import (
	"fmt"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/system"
	"github.com/jumppad-labs/jumppad/pkg/clients/container/mocks"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/clients/tar"
	"github.com/jumppad-labs/jumppad/testutils"
	"github.com/stretchr/testify/mock"
	assert "github.com/stretchr/testify/require"
)

// setupEngineMocks sets the engine type, the security options reported by
// Info, and the oom_score_adj that ContainerInspect reports after a start.
func setupEngineMocks(md *mocks.Docker, engine string, securityOptions []string, inspectedOOM int) {
	testutils.RemoveOn(&md.Mock, "ServerVersion")
	testutils.RemoveOn(&md.Mock, "Info")
	testutils.RemoveOn(&md.Mock, "ContainerInspect")

	md.On("ServerVersion", mock.Anything).Return(types.Version{Components: []types.ComponentVersion{{Name: engine}}}, nil)
	md.On("Info", mock.Anything).Return(system.Info{Driver: StorageDriverOverlay2, SecurityOptions: securityOptions}, nil)
	md.On("ContainerInspect", mock.Anything, mock.Anything).Return(container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{
			State:      &container.State{Running: false},
			HostConfig: &container.HostConfig{OomScoreAdj: inspectedOOM},
		},
		NetworkSettings: &container.NetworkSettings{Networks: map[string]*network.EndpointSettings{"bridge": nil}},
	}, nil)
}

func isProbe(cfg *container.Config) bool {
	return len(cfg.Cmd) == 1 && cfg.Cmd[0] == "true"
}

// createCalls splits the ContainerCreate calls into the probe and the rest
func createCalls(md *mocks.Docker) (probes []mock.Call, others []mock.Call) {
	for _, c := range testutils.GetCalls(&md.Mock, "ContainerCreate") {
		if isProbe(c.Arguments[1].(*container.Config)) {
			probes = append(probes, c)
		} else {
			others = append(others, c)
		}
	}

	return probes, others
}

func TestCreateContainerRootlessPodmanSetsOOMScoreAdjFloor(t *testing.T) {
	cc, md, mic := createContainerConfig()
	setupEngineMocks(md, "Podman Engine", []string{"name=seccomp,profile=default", "name=rootless"}, 200)

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	probes, others := createCalls(md)
	assert.Len(t, probes, 1)
	assert.Len(t, others, 1)

	// the probe asks for the default and has no network
	probeHC := probes[0].Arguments[2].(*container.HostConfig)
	assert.Equal(t, 0, probeHC.OomScoreAdj)
	assert.Equal(t, container.NetworkMode("none"), probeHC.NetworkMode)
	md.AssertCalled(t, "ContainerRemove", mock.Anything, "test", mock.Anything)

	// the container is created with the value Podman gave the probe
	hc := others[0].Arguments[2].(*container.HostConfig)
	assert.Equal(t, 200, hc.OomScoreAdj)
}

func TestCreateContainerRootlessPodmanProbesOnce(t *testing.T) {
	cc, md, mic := createContainerConfig()
	setupEngineMocks(md, "Podman Engine", []string{"name=rootless"}, 200)

	p, err := NewDockerTasks(md, mic, &tar.TarGz{}, logger.NewTestLogger(t))
	assert.NoError(t, err)

	for i := 0; i < 3; i++ {
		cc.Name = fmt.Sprintf("testcontainer%d", i)
		_, err = p.CreateContainer(cc)
		assert.NoError(t, err)
	}

	probes, others := createCalls(md)
	assert.Len(t, probes, 1)
	assert.Len(t, others, 3)

	for _, c := range others {
		assert.Equal(t, 200, c.Arguments[2].(*container.HostConfig).OomScoreAdj)
	}
}

func TestCreateContainerDoesNotSetOOMScoreAdjForDockerOrRootfulPodman(t *testing.T) {
	tests := []struct {
		name            string
		engine          string
		securityOptions []string
	}{
		{"docker", "Engine", []string{"name=seccomp,profile=default"}},
		{"rootless docker", "Engine", []string{"name=rootless"}},
		{"rootful podman", "Podman Engine", []string{"name=seccomp,profile=default"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc, md, mic := createContainerConfig()
			setupEngineMocks(md, tt.engine, tt.securityOptions, 200)

			err := setupContainer(t, cc, md, mic)
			assert.NoError(t, err)

			probes, others := createCalls(md)
			assert.Len(t, probes, 0)
			assert.Len(t, others, 1)
			assert.Equal(t, 0, others[0].Arguments[2].(*container.HostConfig).OomScoreAdj)
		})
	}
}

func TestCreateContainerRootlessPodmanProbeFailureKeepsDefault(t *testing.T) {
	cc, md, mic := createContainerConfig()
	setupEngineMocks(md, "Podman Engine", []string{"name=rootless"}, 200)

	// the probe container gets its own ID, and fails to start
	probeCfg := mock.MatchedBy(func(cfg *container.Config) bool { return isProbe(cfg) })
	md.ExpectedCalls = append([]*mock.Call{
		md.On("ContainerCreate", mock.Anything, probeCfg, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(container.CreateResponse{ID: "probe"}, nil),
		md.On("ContainerStart", mock.Anything, "probe", mock.Anything).Return(fmt.Errorf("boom")),
	}, md.ExpectedCalls[:len(md.ExpectedCalls)-2]...)

	err := setupContainer(t, cc, md, mic)
	assert.NoError(t, err)

	md.AssertCalled(t, "ContainerRemove", mock.Anything, "probe", mock.Anything)

	_, others := createCalls(md)
	assert.Len(t, others, 1)
	assert.Equal(t, 0, others[0].Arguments[2].(*container.HostConfig).OomScoreAdj)
}
