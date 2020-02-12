package config

// TypeExecLocal is the resource string for a LocalExec resource
const TypeExecLocal ResourceType = "exec_local"

// ExecLocal allows commands to be executed on the local machine
type ExecLocal struct {
	ResourceInfo

	// Either Script or Command must be specified
	Script    string   `hcl:"script,optional" json:"script,omitempty"` // Path to a script to execute
	Command   string   `hcl:"cmd,optional" json:"cmd,omitempty"`       // Command to execute
	Arguments []string `hcl:"args,optional" json:"args,omitempty"`     // only used when combined with Command

	Environment []KV `hcl:"env,block" json:"env"` // Envrionment variables to set
}

// NewExecLocal creates a LocalExec resource with the default values
func NewExecLocal(name string) *ExecLocal {
	return &ExecLocal{ResourceInfo: ResourceInfo{Name: name, Type: TypeExecLocal, Status: PendingCreation}}
}
