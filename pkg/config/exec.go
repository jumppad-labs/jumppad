package config

// LocalExec allows commands to be executed on the local machine
type LocalExec struct {
	Name  string
	State State

	// Either Script or Command must be specified
	Script    string   `hcl:"script,optional"` // Path to a script to execute
	Command   string   `hcl:"cmd,optional"`    // Command to execute
	Arguments []string `hcl:"args,optional"`   // only used when combined with Command

	Environment []KV `hcl:"env,block"` // Envrionment variables to set
}

// RemoteExec allows commands to be executed in remote containers
type RemoteExec struct {
	Name  string
	State State

	TargetRef  interface{}
	NetworkRef *Network // Automatically fetched from target
	WANRef     *Network // Automatically created

	// Either Image or Target must be specified
	Image   *Image `hcl:"image,block"`      // Create a new container and exec
	Network string `hcl:"network,optional"` // Attach to the correct network // only when Image is specified

	Target string `hcl:"target,optional"` // Attach to a running target and exec

	// Either Script or Command must be specified
	Script    string   `hcl:"script,optional"` // Path to a script to execute
	Command   string   `hcl:"cmd,optional"`    // Command to execute
	Arguments []string `hcl:"args,optional"`   // only used when combined with Command

	Volumes     []Volume `hcl:"volume,block"` // Volumes to mount to container
	Environment []KV     `hcl:"env,block"`    // Environment varialbes to set
}
