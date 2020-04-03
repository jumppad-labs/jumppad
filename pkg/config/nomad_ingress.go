package config

// TypeNomadIngress is the resource string for the type
const TypeNomadIngress ResourceType = "nomad_ingress"

// NomadIngress defines an ingress service mapping ports between local host or docker network and the target
type NomadIngress struct {
	ResourceInfo

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	Networks []NetworkAttachment `hcl:"network,block" json:"networks,omitempty"` // Attach to the correct network // only when Image is specified

	Cluster string `hcl:"cluster" json:"cluster"`

	Job   string `hcl:"job" json:"job"`
	Group string `hcl:"group" json:"group"`
	Task  string `hcl:"task" json:"task"`

	Ports []Port `hcl:"port,block" json:"ports,omitempty"`
}

// NewNomadIngress creates a new ingress with the correct defaults
func NewNomadIngress(name string) *NomadIngress {
	return &NomadIngress{ResourceInfo: ResourceInfo{Name: name, Type: TypeNomadIngress, Status: PendingCreation}}
}
