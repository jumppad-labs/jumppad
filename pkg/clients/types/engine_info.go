package types

type EngineInfo struct {
	// StorageDriver used by the engine, overlay, devicemapper, etc
	StorageDriver string

	// EngineType, docker, podman, not found
	EngineType string
}
