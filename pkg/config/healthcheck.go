package config

// HealthCheck is an internal block for configuration which
// allows the user to define the criteria for successful creation
// example config:
//    http     		= "http://consul-consul:8500/v1/leader"                          // can the http endpoint be reached
//    tcp      		= "consul-consul:8500"                                           // can a TCP connection be made
//    services 		= ["consul-consul"]                                              // does service exist and there are endpoints
//    pods     		= ["component=server,app=consul", "component=client,app=consul"] // is the pod running and healthy
//    nomad_jobs = ["redis"] 																										   // are the Nomad jobs running and healthy
type HealthCheck struct {
	Timeout   string   `hcl:"timeout" json:"timeout"`
	HTTP      string   `hcl:"http,optional" json:"http,omitempty"`
	TCP       string   `hcl:"tcp,optional" json:"tcp,omitempty"`
	Services  []string `hcl:"services,optional" json:"services,omitempty"`
	Pods      []string `hcl:"pods,optional" json:"pods,omitempty"`
	NomadJobs []string `hcl:"nomad_jobs,optional" json:"nomad_jobs,omitempty" mapstructure:"nomad_jobs"`
}
