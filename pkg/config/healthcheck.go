package config

// HealthCheck is an internal block for configuration which
// allows the user to define the criteria for successful creation
// example config:
//    http     		= "http://consul-consul:8500/v1/leader"                          // can the http endpoint be reached
//    tcp      		= "consul-consul:8500"                                           // can a TCP connection be made
//    services 		= ["consul-consul"]                                              // does service exist and there are endpoints
//    pods     		= ["component=server,app=consul", "component=client,app=consul"] // is the pod running and healthy
//    nomad_tasks = ["redis"] 																										 // are the Nomad tasks running and healthy
type HealthCheck struct {
	Timeout    string   `hcl:"timeout"`
	HTTP       string   `hcl:"http,optional"`
	TCP        string   `hcl:"tcp,optional"`
	Services   []string `hcl:"services,optional"`
	Pods       []string `hcl:"pods,optional"`
	NomadTasks []string `hcl:"nomad_tasks,optional"`
}
