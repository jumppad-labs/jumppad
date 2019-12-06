package config

type Helm struct {
	Name       string
	ClusterRef *Cluster

	Cluster string `hcl:"cluster"`
	Chart   string `hcl:"chart"`
	Values  string `hcl:"values,optional"`

	HealthCheck *HealthCheck `hcl:"health_check,block"`
}
