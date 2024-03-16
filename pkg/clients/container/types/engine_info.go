package types

type EngineInfo struct {
	// StorageDriver used by the engine, overlay, devicemapper, etc
	StorageDriver string

	// EngineType, docker, podman, not found
	EngineType string

	// EngineType, docker, podman, not found
	CPU    int
	Memory int
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
)
