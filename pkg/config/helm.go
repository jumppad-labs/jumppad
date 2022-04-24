package config

// TypeHelm is the string representation of the ResourceType
const TypeHelm ResourceType = "helm"

// Helm defines configuration for running Helm charts
type Helm struct {
	ResourceInfo `hcl:",remain" mapstructure:",squash"`

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	Cluster string `hcl:"cluster" json:"cluster"`

	// Optional HelmRepository, if specified will try to download the chart from the give repository
	Repository *HelmRepository `hcl:"repository,block" json:"repository"`

	// name of the chart within the repository or Go Getter reference to download chart from
	Chart string `hcl:"chart" json:"chart"`

	// name to use when installing the chart, if blank uses resource name
	ChartName string `hcl:"chart_name,optional" json:"chart_name,omitempty" mapstructure:"chart_name"`

	// semver of the chart to install
	Version string `hcl:"version,optional" json:"version,omitempty"`

	Values       string            `hcl:"values,optional" json:"values"`
	ValuesString map[string]string `hcl:"values_string,optional" json:"values_string" mapstructure:"values_string"`

	// Namespace is the Kubernetes namespace
	Namespace string `hcl:"namespace,optional" json:"namespace,omitempty"`

	// CreateNamespace when set to true Helm wiil creeate the namespace before installing
	CreateNamespace bool `hcl:"create_namespace,optional" json:"create_namespace,omitempty" mapstructure:"create_namespace"`

	// Skip the install of any CRDs
	SkipCRDs bool `hcl:"skip_crds,optional" json:"skip_crds,omitempty" mapstructure:"skip_crds"`

	// Retry the install n number of times
	Retry int `hcl:"retry,optional" json:"retry,omitempty" mapstructure:"retry"`

	HealthCheck *HealthCheck `hcl:"health_check,block" json:"health_check,omitempty" mapstructure:"health_check"`
}

type HelmRepository struct {
	Name string `hcl:"name" json:"name"`
	URL  string `hcl:"url" json:"url"`
}

// NewHelm creates a new Helm resource with the correct defaults
func NewHelm(name string) *Helm {
	return &Helm{ResourceInfo: ResourceInfo{Name: name, Type: TypeHelm, Status: PendingCreation}}
}
