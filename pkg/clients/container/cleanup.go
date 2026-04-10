package container

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
)

// cleanupImage is the image used by CleanupHostPath to remove a host path.
const cleanupImage = "alpine:latest"

// CleanupHostPath removes hostPath (the folder itself and everything inside
// it) by running a short lived alpine container that bind mounts the parent
// of hostPath at /parent and deletes the target as the container root user.
//
// This is required because Jumppad managed containers (for example Vault) may
// write files into a mounted data folder using a UID that does not belong to
// the host user, which causes host side os.RemoveAll to fail with EPERM. The
// parent of hostPath is mounted, rather than hostPath itself, so the target
// folder can be unlinked from inside the container — a bind mount point
// cannot remove itself.
//
// TODO: this mirrors the previous host side behaviour of wiping the entire
// data folder. A future version should only remove folders referenced by the
// current config.
func CleanupHostPath(ct ContainerTasks, l logger.Logger, hostPath string) error {
	parent := filepath.Dir(hostPath)
	target := filepath.Base(hostPath)

	if err := ct.PullImage(types.Image{Name: cleanupImage}, false); err != nil {
		return fmt.Errorf("unable to pull %s for host path cleanup: %w", cleanupImage, err)
	}

	cc := &types.Container{
		Name:  fmt.Sprintf("jumppad-cleanup-%d", time.Now().UnixNano()),
		Image: &types.Image{Name: cleanupImage},
		Volumes: []types.Volume{
			{
				Source:      parent,
				Destination: "/parent",
				Type:        "bind",
			},
		},
		Command: []string{"tail", "-f", "/dev/null"},
	}

	id, err := ct.CreateContainer(cc)
	if err != nil {
		return fmt.Errorf("unable to create cleanup container: %w", err)
	}
	defer func() {
		if rerr := ct.RemoveContainer(id, true); rerr != nil {
			l.Warn("Unable to remove cleanup container", "id", id, "error", rerr)
		}
	}()

	// Wait for the container to be ready to accept an exec. ContainerInfo
	// returns an opaque interface so the simplest portable readiness check is
	// to retry a trivial command until it succeeds.
	var readyErr error
	for range 10 {
		_, readyErr = ct.ExecuteCommand(id, []string{"true"}, nil, "/", "", "", 10, nil)
		if readyErr == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if readyErr != nil {
		return fmt.Errorf("cleanup container did not become ready: %w", readyErr)
	}

	if _, err := ct.ExecuteCommand(
		id,
		[]string{"rm", "-rf", filepath.ToSlash(filepath.Join("/parent", target))},
		nil, "/", "", "", 300, nil,
	); err != nil {
		return fmt.Errorf("unable to remove %s via cleanup container: %w", hostPath, err)
	}

	l.Debug("Cleaned up host path via container", "path", hostPath)
	return nil
}
