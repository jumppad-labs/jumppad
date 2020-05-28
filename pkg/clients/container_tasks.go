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
	SetForcePull(bool)
	// CreateContainer creates a new container for the given configuration
	// if successful CreateContainer returns the ID of the created container and a nil error
	// if not successful CreateContainer returns a blank string for the id and an error message
	CreateContainer(*config.Container) (id string, err error)
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
	FindContainerIDs(name string, typeName config.ResourceType) ([]string, error)
	// ContainerLogs attaches to the container and streams the logs to the returned
	// io.ReadCloser.
	// Returns an error if the container is not running
	ContainerLogs(id string, stdOut, stdErr bool) (io.ReadCloser, error)
	// CopyFromContainer allows the copying of a file from a container
	CopyFromContainer(id, src, dst string) error
	// CopyLocaDockerImageToVolume copies the docker images to the docker volume as a
	// compressed archive.
	// the path in the docker volume where the archive is created is returned
	// along with any errors.
	CopyLocalDockerImageToVolume(images []string, volume string, force bool) ([]string, error)
	// Execute command allows the execution of commands in a running docker container
	// id is the id of the container to execute the command in
	// command is a slice of strings to execute
	// writer [optional] will be used to write any output from the command execution.
	ExecuteCommand(id string, command []string, env []string, workingDirectory string, writer io.Writer) error
	// NetworkDisconnect disconnects a container from the network
	DetachNetwork(network, containerid string) error

	// CreateShell in the running container and attach
	CreateShell(id string, command []string, stdin io.ReadCloser, stdout io.Writer, stderr io.Writer) error
}
