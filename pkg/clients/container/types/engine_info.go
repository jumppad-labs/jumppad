package types

type EngineInfo struct {
	// StorageDriver used by the engine, overlay, devicemapper, etc
	StorageDriver string

	// EngineType, docker, podman, not found
	EngineType string

	// EngineType, docker, podman, not found
	CPU    int
	Memory int

	// Rootless is true when the engine runs without root, containers run in a
	// user namespace and can't access host devices such as /dev/kmsg
	Rootless bool
}

const (
	EngineTypeDocker = "docker"
	EngineTypePodman = "podman"
	EngineNotFound   = "not found"
)

const (
	StorageDriverOverlay2     = "overlay2"
	StorageDriverFuse         = "fuse-overlayfs"
	StorageDriverBTRFS        = "btrfs"
	StorageDriverZFS          = "zfs"
	StorageDriverVFS          = "vfs"
	StorageDriverAUFS         = "aufs"
	StorageDriverDeviceMapper = "devicemapper"
	StorageDriverOverlay      = "overlay"
	// StorageDriverOverlayFS is reported by Docker's containerd image store
	// (the default on new Docker Desktop installs)
	StorageDriverOverlayFS = "overlayfs"
)
