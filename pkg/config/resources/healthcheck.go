package resources

// HealthCheckContainer is an internal block for configuration which
// allows the user to define the criteria for successful creation
type HealthCheckContainer struct {
	// Timeout expressed as a go duration i.e 10s
	Timeout string `hcl:"timeout" json:"timeout"`

	HTTP []HealthCheckHTTP `hcl:"http,block" json:"http,omitempty"`
	TCP  []HealthCheckTCP  `hcl:"tcp,block" json:"tcp,omitempty"`
	Exec []HealthCheckExec `hcl:"exec,block" json:"exec,omitempty"`
}

// HealthCheckHTTP defines a HTTP based health check
type HealthCheckHTTP struct {
	Address      string              `hcl:"address" json:"address,omitempty"`                      // HTTP endpoint to check
	Method       string              `hcl:"method,optional" json:"method,omitempty"`               // HTTP method to use, default GET
	Body         string              `hcl:"body,optional" json:"body,omitempty"`                   // Payload to send with check
	Headers      map[string][]string `hcl:"headers,optional" json:"headers,omitempty"`             // HTTP headers to send with request
	SuccessCodes []int               `hcl:"success_codes,optional" json:"success_codes,omitempty"` // HTTP status codes that signal the health of the endpoint, default 200
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
	// ExitCode to mark a successful check, default 0
	ExitCode int `hcl:"exit_code,optional" json:"exit_code,omitempty"`

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
