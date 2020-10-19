package config

// TypeExecLocal is the resource string for a LocalExec resource
const TypeExecLocal ResourceType = "exec_local"

// ExecLocal allows commands to be executed on the local machine
type ExecLocal struct {
	ResourceInfo

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	Command          string   `hcl:"cmd,optional" json:"cmd,omitempty"`                             // Command to execute
	Arguments        []string `hcl:"args,optional" json:"args,omitempty"`                           // only used when combined with Command
	WorkingDirectory string   `hcl:"working_directory,optional" json:"working_directory,omitempty"` // Working directory to execute commands

	Environment []KV              `hcl:"env,block" json:"env"`                      // environment variables to set
	EnvVar      map[string]string `hcl:"env_var,optional" json:"env_var,omitempty"` // environment variables to set
}

// NewExecLocal creates a LocalExec resource with the default values
func NewExecLocal(name string) *ExecLocal {
	return &ExecLocal{ResourceInfo: ResourceInfo{Name: name, Type: TypeExecLocal, Status: PendingCreation}}
}
