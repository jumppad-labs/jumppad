package clients

import (
	"io"

	"github.com/shipyard-run/shipyard/pkg/config"
)

// ContainerTasks is a task oriented client which abstracts
// the underlying container technology from the providers
// this allows different concrete implementations such as Docker, or ContainerD
// without needing to change the provider code.
//
// The Docker SDK can also be quite terse, the API design for this client
// is design is centered around performing a task such as CreateContainer,
// this may be composed of many individual SDK calls.
type ContainerTasks interface {
	// CreateContainer creates a new container for the given configuration
	// if successful CreateContainer returns the ID of the created container and a nil error
	// if not successful CreateContainer returns a blank string for the id and an error message
	CreateContainer(config.Container) (id string, err error)
	// RemoveContainer stops and removes a running container
	RemoveContainer(id string) error
	// CreateVolume creates a new volume with the given name.
	// If successful the id of the newly created volume is returned
	CreateVolume(name string) (id string, err error)
	// RemoveVolume removes a volume with the given name
	RemoveVolume(name string) error
	// PullImage pulls a Docker image from the registry if it is not already
	// present in the local cache.
	// If the Username and Password config options are set then PullImage will attempt to
	// authenticate with the registry before pulling the image.
	// If the force parameter is set then PullImage will pull regardless of the image already
	// being cached locally.
	PullImage(image config.Image, force bool) error
	// FindContainerIDs returns the Container IDs for the given identifier
	FindContainerIDs(name string, networkName string) ([]string, error)
	// ContainerLogs attaches to the container and streams the logs to the returned
	// io.ReadCloser.
	// Returns an error if the container is not running
	ContainerLogs(id string, stdOut, stdErr bool) (io.ReadCloser, error)
}
