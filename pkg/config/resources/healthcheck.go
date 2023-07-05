package resources

// HealthCheckContainer is an internal block for configuration which
// allows the user to define the criteria for successful creation
type HealthCheckContainer struct {
	// Timeout expressed as a go duration i.e 10s
	Timeout string `hcl:"timeout" json:"timeout"`

	HTTP *HealthCheckHTTP `hcl:"http,block" json:"http,omitempty"`
	TCP  *HealthCheckTCP  `hcl:"tcp,block" json:"tcp,omitempty"`
	Exec *HealthCheckExec `hcl:"exec,block" json:"exec,omitempty"`
}

// HealthCheckHTTP defines a HTTP based health check
type HealthCheckHTTP struct {
	// address = "http://consul-consul:8500/v1/leader" // can the http endpoint be reached
	Address string `hcl:"address" json:"address,omitempty"`
	// success_codes  = [200,429] // https status codes that signal the health of the endpoint
	SuccessCodes []int `hcl:"success_codes" json:"success_codes,omitempty"`
}

type HealthCheckTCP struct {
	// address = "consul-consul:8500" // can a TCP connection be made
	Address string `hcl:"address" json:"address,omitempty"`
}

type HealthCheckExec struct {
	// Command to execute, the command is run in the target container
	Command []string `hcl:"command,optional" json:"command,omitempty"`
	// Script specified as a string to execute, the script can be a bash or a sh script
	// scripts are copied to the container /tmp directory, marked as executable and run
	Script string `hcl:"script,optional" json:"script,omitempty"`
}

type HealthCheckKubernetes struct {
	// Timeout expressed as a go duration i.e 10s
	Timeout string `hcl:"timeout" json:"timeout"`
	//	pods = ["component=server,app=consul", "component=client,app=consul"] // is the pod running and healthy
	Pods []string `hcl:"pods" json:"pods,omitempty"`
}

type HealthCheckNomad struct {
	// Timeout expressed as a go duration i.e 10s
	Timeout string `hcl:"timeout" json:"timeout"`
	//	jobs = ["redis"] // are the Nomad jobs running and healthy
	Jobs []string `hcl:"jobs" json:"jobs,omitempty"`
}
