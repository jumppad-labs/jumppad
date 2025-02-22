package exec

import (
	"fmt"
	"strings"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	ctypes "github.com/jumppad-labs/jumppad/pkg/config/resources/container"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/zclconf/go-cty/cty"
)

// TypeExec is the resource string for an Exec resource
const TypeExec string = "exec"

// Exec allows commands to be executed either locally or remotely
type Exec struct {
	// embedded type holding name, etc
	types.ResourceBase `hcl:",remain"`

	Script           string            `hcl:"script" json:"script"`                                          // script to execute
	WorkingDirectory string            `hcl:"working_directory,optional" json:"working_directory,omitempty"` // Working directory to execute commands
	Daemon           bool              `hcl:"daemon,optional" json:"daemon,omitempty"`                       // Should the process run as a daemon
	Timeout          string            `hcl:"timeout,optional" json:"timeout,omitempty"`                     // Set the timeout for the command
	Environment      map[string]string `hcl:"environment,optional" json:"environment,omitempty"`             // environment variables to set

	// If remote, either Image or Target must be specified
	Image  *ctypes.Image     `hcl:"image,block" json:"image,omitempty"`      // Create a new container and exec
	Target *ctypes.Container `hcl:"target,optional" json:"target,omitempty"` // Attach to a running target and exec

	Networks []ctypes.NetworkAttachment `hcl:"network,block" json:"networks,omitempty"` // Attach to the correct network // only when Image is specified
	Volumes  []ctypes.Volume            `hcl:"volume,block" json:"volumes,omitempty"`   // Volumes to mount to container
	RunAs    *ctypes.User               `hcl:"run_as,block" json:"run_as,omitempty"`    // User block for mapping the user id and group id inside the container

	// output
	PID      int       `hcl:"pid,optional" json:"pid,omitempty"`             // PID stores the ID of the created connector service if it is a local exec
	ExitCode int       `hcl:"exit_code,optional" json:"exit_code,omitempty"` // Exit code of the process
	Output   cty.Value `hcl:"output,optional"`                               // output values returned from exec
}

func (e *Exec) Process() error {
	// check if it is a remote exec
	if e.Image != nil || e.Target != nil {
		// process volumes
		// make sure mount paths are absolute
		for i, v := range e.Volumes {
			e.Volumes[i].Source = utils.EnsureAbsolute(v.Source, e.Meta.File)
		}

		// make sure line endings are linux
		e.Script = strings.Replace(e.Script, "\r\n", "\n", -1)
	} else {
		if len(e.Networks) > 0 || len(e.Volumes) > 0 {
			return fmt.Errorf("unable to create local exec with networks or volumes")
		}
	}

	// e.Output = map[string]string{}

	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(e.Meta.ID)

		if r != nil {
			kstate := r.(*Exec)
			e.PID = kstate.PID
			e.ExitCode = kstate.ExitCode
			e.Output = kstate.Output
		}
	}

	return nil
}
