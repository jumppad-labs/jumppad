package config

// TypeHelm is the string representation of the ResourceType
const TypeHelm ResourceType = "helm"

// Helm defines configuration for running Helm charts
type Helm struct {
	ResourceInfo `hcl:",remain" mapstructure:",squash"`

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	Cluster      string            `hcl:"cluster" json:"cluster"`
	Chart        string            `hcl:"chart" json:"chart"`
	Values       string            `hcl:"values,optional" json:"values"`
	ValuesString map[string]string `hcl:"values_string,optional" json:"values_string" mapstructure:"values_string"`

	// ChartName is the name of the chart, if not present
	// uses the name of the resource block
	ChartName string `hcl:"chart_name,optional" json:"chart_name,omitempty" mapstructure:"chart_name"`

	// Namespace is the Kubernetes namespace
	Namespace string `hcl:"namespace,optional" json:"namespace,omitempty"`

	// CreateNamespace when set to true Helm wiil creeate the namespace before installing
	CreateNamespace bool `hcl:"create_namespace,optional" json:"create_namespace,omitempty" mapstructure:"create_namespace"`

	HealthCheck *HealthCheck `hcl:"health_check,block" json:"health_check,omitempty" mapstructure:"health_check"`
}

// NewHelm creates a new Helm resource with the correct detaults
func NewHelm(name string) *Helm {
	return &Helm{ResourceInfo: ResourceInfo{Name: name, Type: TypeHelm, Status: PendingCreation}}
}
