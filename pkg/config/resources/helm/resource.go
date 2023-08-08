package helm

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/healthcheck"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/k8s"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// TypeHelm is the string representation of the ResourceType
const TypeHelm string = "helm"

// Helm defines configuration for running Helm charts
type Helm struct {
	types.ResourceMetadata `hcl:",remain"`

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	Cluster k8s.K8sCluster `hcl:"cluster" json:"cluster"`

	// Optional HelmRepository, if specified will try to download the chart from the give repository
	Repository *HelmRepository `hcl:"repository,block" json:"repository"`

	// name of the chart within the repository or Go Getter reference to download chart from
	Chart string `hcl:"chart" json:"chart"`

	// semver of the chart to install
	Version string `hcl:"version,optional" json:"version,omitempty"`

	Values       string            `hcl:"values,optional" json:"values"`
	ValuesString map[string]string `hcl:"values_string,optional" json:"values_string"`

	// Namespace is the Kubernetes namespace
	Namespace string `hcl:"namespace,optional" json:"namespace,omitempty"`

	// CreateNamespace when set to true Helm will create the namespace before installing
	CreateNamespace bool `hcl:"create_namespace,optional" json:"create_namespace,omitempty"`

	// Skip the install of any CRDs
	SkipCRDs bool `hcl:"skip_crds,optional" json:"skip_crds,omitempty"`

	// Retry the install n number of times
	Retry int `hcl:"retry,optional" json:"retry,omitempty"`

	// Timeout specifies the maximum time a chart can run, default 300s
	Timeout string `hcl:"timeout,optional" json:"timeout"`

	// Define health checks for the pods deployed by the chart
	HealthCheck *healthcheck.HealthCheckKubernetes `hcl:"health_check,block" json:"health_check,omitempty"`
}

type HelmRepository struct {
	Name string `hcl:"name" json:"name"`
	URL  string `hcl:"url" json:"url"`
}

func (h *Helm) Process() error {
	// only set absolute if is local folder
	if h.Chart != "" && utils.IsLocalFolder(utils.EnsureAbsolute(h.Chart, h.File)) {
		h.Chart = utils.EnsureAbsolute(h.Chart, h.File)
	}

	if h.Values != "" {
		h.Values = utils.EnsureAbsolute(h.Values, h.File)
	}

	return nil
}
