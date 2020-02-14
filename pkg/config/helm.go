package config

// TypeHelm is the string representation of the ResourceType
const TypeHelm ResourceType = "helm"

// Helm defines configuration for running Helm charts
type Helm struct {
	ResourceInfo

	Cluster string `hcl:"cluster"`
	Chart   string `hcl:"chart"`
	Values  string `hcl:"values,optional"`

	HealthCheck *HealthCheck `hcl:"health_check,block"`
}

// NewHelm creates a new Helm resource with the correct detaults
func NewHelm(name string) *Helm {
	return &Helm{ResourceInfo: ResourceInfo{Name: name, Type: TypeHelm, Status: PendingCreation}}
}
