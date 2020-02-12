package config

// TypeIngress is the resource string for the type
const TypeIngress ResourceType = "ingress"

// Ingress defines an ingress service mapping ports between local host or docker network and the target
type Ingress struct {
	ResourceInfo

	TargetRef  interface{}
	NetworkRef *Network // automatically fetched from target
	WANRef     *Network // automatically created

	Target    string `hcl:"target"`
	Service   string `hcl:"service,optional"`
	Namespace string `hcl:"namespace,optional"`
	Ports     []Port `hcl:"port,block"`
	IPAddress string `hcl:"ip_address,optional"`
}

// NewIngress creates a new ingress with the correct defaults
func NewIngress(name string) *Ingress {
	return &Ingress{ResourceInfo: ResourceInfo{Name: name, Type: TypeIngress, Status: PendingCreation}}
}
