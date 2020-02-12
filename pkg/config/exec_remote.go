package config

// TypeExecRemote is the resource string for a ExecRemote resource
const TypeExecRemote ResourceType = "exec_remote"

// ExecRemote allows commands to be executed in remote containers
type ExecRemote struct {
	ResourceInfo

	// Either Image or Target must be specified
	Image   *Image `hcl:"image,block" json:"image,omitempty"`        // Create a new container and exec
	Network string `hcl:"network,optional" json:"network,omitempty"` // Attach to the correct network // only when Image is specified

	Target string `hcl:"target,optional" json:"target,omitempty"` // Attach to a running target and exec

	// Either Script or Command must be specified
	Script    string   `hcl:"script,optional" json:"script,omitempty"` // Path to a script to execute
	Command   string   `hcl:"cmd,optional" json:"cmd,omitempty"`       // Command to execute
	Arguments []string `hcl:"args,optional" json:"args,omitempty"`     // only used when combined with Command

	Volumes     []Volume `hcl:"volume,block" json:"volumes,omitempty"` // Volumes to mount to container
	Environment []KV     `hcl:"env,block" json:"env,omitempty"`        // Environment varialbes to set
}

// NewExecRemote creates a ExecRemote resorurce with the detault values
func NewExecRemote(name string) *ExecRemote {
	return &ExecRemote{ResourceInfo: ResourceInfo{Name: name, Type: TypeExecRemote, Status: PendingCreation}}
}
