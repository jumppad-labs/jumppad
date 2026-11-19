package container

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	dtypes "github.com/jumppad-labs/jumppad/pkg/clients/container/types"
)

// oomScoreAdjFloor returns the lowest oom_score_adj that new containers can
// be given, or 0 when there is no floor.
//
// The Docker SDK always sends HostConfig.OomScoreAdj, so every container is
// created with an explicit 0. Rootless Podman can't lower a process's
// oom_score_adj below its own (usually 200 for a systemd user service). It
// raises the container's value to its own when the container starts, but it
// writes the requested 0 into the spec of every exec. Since crun 1.31 applies
// the exec's oomScoreAdj, every exec then fails with exit code 126:
//
//	crun: write to `/proc/self/oom_score_adj`: Permission denied
//
// Creating containers with the floor keeps the container and exec specs the
// same. The floor is found once by starting a throwaway container and reading
// back the value Podman gave it. Docker and rootful Podman have no floor.
func (d *DockerTasks) oomScoreAdjFloor() int {
	if d.engineType != dtypes.EngineTypePodman || !d.rootless {
		return 0
	}

	d.oomFloorOnce.Do(func() {
		floor, err := d.probeOOMScoreAdjFloor()
		if err != nil {
			d.l.Warn("Unable to determine the oom_score_adj of rootless Podman, exec in containers may fail", "error", err)
			return
		}

		d.l.Debug("Rootless Podman, creating containers with its oom_score_adj", "oom_score_adj", floor)
		d.oomFloor = floor
	})

	return d.oomFloor
}

// probeOOMScoreAdjFloor starts a container that asks for oom_score_adj 0 and
// returns the value the engine actually gave it.
func (d *DockerTasks) probeOOMScoreAdjFloor() (int, error) {
	img := dtypes.Image{Name: "alpine:latest"}

	err := d.PullImage(img, false)
	if err != nil {
		return 0, fmt.Errorf("unable to pull '%s': %w", img.Name, err)
	}

	ctx := context.Background()

	cont, err := d.c.ContainerCreate(
		ctx,
		&container.Config{Image: makeImageCanonical(img.Name), Cmd: []string{"true"}},
		&container.HostConfig{NetworkMode: "none"},
		nil,
		nil,
		"",
	)
	if err != nil {
		return 0, fmt.Errorf("unable to create container: %w", err)
	}

	defer d.c.ContainerRemove(ctx, cont.ID, container.RemoveOptions{Force: true})

	err = d.c.ContainerStart(ctx, cont.ID, container.StartOptions{})
	if err != nil {
		return 0, fmt.Errorf("unable to start container: %w", err)
	}

	// the value is adjusted when the container starts, wait for it to exit so
	// the inspected value is final
	for i := 0; i < 50; i++ {
		info, err := d.c.ContainerInspect(ctx, cont.ID)
		if err != nil {
			return 0, fmt.Errorf("unable to inspect container: %w", err)
		}

		if info.State == nil || !info.State.Running {
			if info.HostConfig == nil {
				return 0, fmt.Errorf("container has no host config")
			}

			return info.HostConfig.OomScoreAdj, nil
		}

		time.Sleep(100 * time.Millisecond)
	}

	return 0, fmt.Errorf("timeout waiting for container to exit")
}
