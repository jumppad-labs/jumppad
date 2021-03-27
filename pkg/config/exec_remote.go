package config

// TypeExecRemote is the resource string for a ExecRemote resource
const TypeExecRemote ResourceType = "exec_remote"

// ExecRemote allows commands to be executed in remote containers
type ExecRemote struct {
	ResourceInfo `hcl:",remain" mapstructure:",squash"`

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	Networks []NetworkAttachment `hcl:"network,block" json:"networks,omitempty"` // Attach to the correct network // only when Image is specified

	// Either Image or Target must be specified
	Image  *Image `hcl:"image,block" json:"image,omitempty"`      // Create a new container and exec
	Target string `hcl:"target,optional" json:"target,omitempty"` // Attach to a running target and exec

	// Either Script or Command must be specified
	//Script    string   `hcl:"script,optional" json:"script,omitempty"` // Path to a script to execute
	Command          string   `hcl:"cmd,optional" json:"cmd,omitempty" mapstructure:"cmd"`                                           // Command to execute
	Arguments        []string `hcl:"args,optional" json:"args,omitempty" mapstructure:"args"`                                        // only used when combined with Command
	WorkingDirectory string   `hcl:"working_directory,optional" json:"working_directory,omitempty" mapstructure:"working_directory"` // Working directory to execute commands

	Volumes     []Volume          `hcl:"volume,block" json:"volumes,omitempty"`                            // Volumes to mount to container
	Environment []KV              `hcl:"env,block" json:"env,omitempty" mapstructure:"env"`                // Environment varialbes to set
	EnvVar      map[string]string `hcl:"env_var,optional" json:"env_var,omitempty" mapstructure:"env_var"` // environment variables to set when starting the container
}

// NewExecRemote creates a ExecRemote resorurce with the detault values
func NewExecRemote(name string) *ExecRemote {
	return &ExecRemote{ResourceInfo: ResourceInfo{Name: name, Type: TypeExecRemote, Status: PendingCreation}}
}
