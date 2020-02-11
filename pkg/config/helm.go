package config

// Helm defines configuration for running Helm charts
type Helm struct {
	Name       string
	State      State
	ClusterRef *Cluster

	Cluster string `hcl:"cluster"`
	Chart   string `hcl:"chart"`
	Values  string `hcl:"values,optional"`

	HealthCheck *HealthCheck `hcl:"health_check,block"`
}
