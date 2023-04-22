package resources

import "github.com/shipyard-run/hclconfig/types"

// TypeExecLocal is the resource string for a LocalExec resource
const TypeLocalExec string = "local_exec"

// ExecLocal allows commands to be executed on the local machine
type LocalExec struct {
	// embedded type holding name, etc
	types.ResourceMetadata `hcl:",remain"`

	Command          []string          `hcl:"cmd,optional" json:"cmd,omitempty"`                             // Command to execute
	WorkingDirectory string            `hcl:"working_directory,optional" json:"working_directory,omitempty"` // Working directory to execute commands
	Daemon           bool              `hcl:"daemon,optional" json:"daemon,omitempty"`                       // Should the process run as a daemon
	Timeout          string            `hcl:"timeout,optional" json:"timeout,omitempty"`                     // Set the timeout for the command
	Environment      map[string]string `hcl:"environment,optional" json:"environment,omitempty"`             // environment variables to set

	// output

	// Pid stores the ID of the created connector service
	Pid int `hcl:"pid,optional" json:"pid,omitempty"`
}

func (e *LocalExec) Process() error {
	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(e.ID)
		if r != nil {
			kstate := r.(*LocalExec)
			e.Pid = kstate.Pid
		}
	}

	return nil
}
