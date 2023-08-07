package resources

import (
	"strings"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/container"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// TypeExecRemote is the resource string for a ExecRemote resource
const TypeRemoteExec string = "remote_exec"

// ExecRemote allows commands to be executed in remote containers
type RemoteExec struct {
	types.ResourceMetadata `hcl:",remain"`

	Networks []container.NetworkAttachment `hcl:"network,block" json:"networks,omitempty"` // Attach to the correct network // only when Image is specified

	// Either Image or Target must be specified
	Image  *container.Image `hcl:"image,block" json:"image,omitempty"`      // Create a new container and exec
	Target string           `hcl:"target,optional" json:"target,omitempty"` // Attach to a running target and exec

	// Either Script or Command must be specified
	Script           string `hcl:"script,optional" json:"script,omitempty"`                       // Path to a script to execute
	WorkingDirectory string `hcl:"working_directory,optional" json:"working_directory,omitempty"` // Working directory to execute commands

	Volumes     []container.Volume `hcl:"volume,block" json:"volumes,omitempty"`             // Volumes to mount to container
	Environment map[string]string  `hcl:"environment,optional" json:"environment,omitempty"` // environment variables to set when starting the container

	// User block for mapping the user id and group id inside the container
	RunAs *container.User `hcl:"run_as,block" json:"run_as,omitempty"`
}

func (e *RemoteExec) Process() error {
	// process volumes
	// make sure mount paths are absolute
	for i, v := range e.Volumes {
		e.Volumes[i].Source = utils.EnsureAbsolute(v.Source, e.File)
	}

	// make sure line endings are linux
	e.Script = strings.Replace(e.Script, "\r\n", "\n", -1)

	return nil
}
