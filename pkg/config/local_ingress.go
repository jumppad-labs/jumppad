package config

// TypeLocalIngress is the resource string for the type
const TypeLocalIngress ResourceType = "local_ingress"

// LocalIngress defines an ingress that exposes an application or service
// running on the local machine to a remote host
type LocalIngress struct {
	ResourceInfo `mapstructure:",squash"`

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	// Cluster to connect to
	Target string `hcl:"target" json:"target"`

	// Destination host or IP address which is acccesible from the
	// local machine
	Destination string `hcl:"destination,optional" json:"destination,omitempty"`

	Ports []Port `hcl:"port,block" json:"ports,omitempty"`

	// id of the created resouce
	Id string `json:"id,omitempty"`
}

// NewK8sIngress creates a new ingress with the correct defaults
func NewLocalIngress(name string) *LocalIngress {
	return &LocalIngress{ResourceInfo: ResourceInfo{Name: name, Type: TypeLocalIngress, Status: PendingCreation}}
}
