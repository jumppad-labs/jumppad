package helm

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/healthcheck"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/k8s"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// TypeHelm is the string representation of the Meta.Type
const TypeHelm string = "helm"

/*
The `helm` resource allows Helm charts to be provisioned to k8s_cluster resources.

@resource
*/
type Helm struct {
	/*
	 embedded type holding name, etc

	 @ignore
	*/
	types.ResourceBase `hcl:",remain"`

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	/*
		A reference to a kubernetes clusters to apply the chart to.
		The system waits until the referenced cluster is healthy before attempting t apply any charts.

		@example
		```
		resource "helm" "consul" {
			cluster = resource.k8s_cluster.k3s
		}
		```
	*/
	Cluster k8s.Cluster `hcl:"cluster" json:"cluster"`
	/*
		The details for the Helm chart repository where the chart exists. I
		f this property is not specifed, the chart location is assumed to be either a local directory or Git reference.
	*/
	Repository *HelmRepository `hcl:"repository,block" json:"repository"`
	/*
		The name of the chart within the repository, or a souce such as a git repository, URL, or file path where the chart file exist.
	*/
	Chart string `hcl:"chart" json:"chart"`
	/*
		Semver of the chart to install, only used when `repository` is specified.
	*/
	Version string `hcl:"version,optional" json:"version,omitempty"`
	/*
		File path to a valid Helm values file to be used when applying the config.

		@example
		```
		resource "helm" "mychart" {
			values = "./values.yaml"
		}
		```
	*/
	Values string `hcl:"values,optional" json:"values"`
	/*
		Map containing helm values to apply with the chart.

		@example
		```
		resource "helm" "mychart" {
			values_string = {
				"global.storage" = "128Mb"
			}
		}
		```
	*/
	ValuesString map[string]string `hcl:"values_string,optional" json:"values_string"`
	// Kubernetes namespace to apply the chart to.
	Namespace string `hcl:"namespace,optional" json:"namespace,omitempty"`
	// If the namespace does not exist, should the helm resource attempt to create it.
	CreateNamespace bool `hcl:"create_namespace,optional" json:"create_namespace,omitempty"`
	// If the chart defines custom resource definitions, should these be ignored.
	SkipCRDs bool `hcl:"skip_crds,optional" json:"skip_crds,omitempty"`
	// Enables the ability to retry the installation of a chart.
	Retry int `hcl:"retry,optional" json:"retry,omitempty"`
	/*
		Maximum time the application phase of a chart can run before failing.
		This duration is different to the health_check that runs after a chart has been applied.
	*/
	Timeout string `hcl:"timeout,optional" json:"timeout"`
	// Health check to run after installing the chart.
	HealthCheck *healthcheck.HealthCheckKubernetes `hcl:"health_check,block" json:"health_check,omitempty"`
}

/*
A `helm_repository` stanza defines the details for a remote helm repository.
*/
type HelmRepository struct {
	// The name of the repository.
	Name string `hcl:"name" json:"name"`
	// The repository URL.
	URL string `hcl:"url" json:"url"`
}

func (h *Helm) Process() error {
	// only set absolute if is local folder
	if h.Chart != "" && utils.IsLocalFolder(utils.EnsureAbsolute(h.Chart, h.Meta.File)) {
		h.Chart = utils.EnsureAbsolute(h.Chart, h.Meta.File)
	}

	if h.Values != "" {
		h.Values = utils.EnsureAbsolute(h.Values, h.Meta.File)
	}

	return nil
}
