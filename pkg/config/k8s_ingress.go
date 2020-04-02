package config

// TypeK8sIngress is the resource string for the type
const TypeK8sIngress ResourceType = "k8s_ingress"

// K8sIngress defines an ingress service mapping ports between local host or docker network and the target
type K8sIngress struct {
	ResourceInfo

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	Networks []NetworkAttachment `hcl:"network,block" json:"networks,omitempty"` // Attach to the correct network // only when Image is specified

	// Cluster to connect to
	Cluster string `hcl:"cluster" json:"cluster"`

	// K8sIngress support the following Kuberentes types
	// which can be connected to:
	// * Service
	// * Deployment
	// * Pod
	// Only one option can be specified at any time

	// Service to proxy to.
	// When proxying to a Kubernetes service, ingress will choose a random
	// pod within that service.
	Service string `hcl:"service,optional" json:"service,omitempty"`
	// Deployment to proxy to.
	// When proxying to a Kubernetes deployment, ingress will choose a random
	// pod within that service.
	Deployment string `hcl:"deployment,optional" json:"deployment,omitempty"`
	// Pod to proxy to
	Pod string `hcl:"pod,optional" json:"pod,omitempty"`

	// Namespace is the Kubernetes namespace
	Namespace string `hcl:"namespace,optional" json:"namespace,omitempty"`

	Ports []Port `hcl:"port,block" json:"ports,omitempty"`
}

// NewK8sIngress creates a new ingress with the correct defaults
func NewK8sIngress(name string) *K8sIngress {
	return &K8sIngress{ResourceInfo: ResourceInfo{Name: name, Type: TypeK8sIngress, Status: PendingCreation}}
}
