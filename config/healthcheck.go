package config

type HealthCheck struct {
	HTTP     string   `hcl:"http,optional"`
	TCP      string   `hcl:"tcp,optional"`
	Services []string `hcl:"services,optional"`
	Pods     []string `hcl:"pods,optional"`
}
