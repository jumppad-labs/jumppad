package config

// TypeContainerIngress is the resource string for the type
const TypeContainerIngress ResourceType = "container_ingress"

// ContainerIngress defines an ingress service mapping ports between local host or docker network and the target
type ContainerIngress struct {
	ResourceInfo `hcl:",remain" mapstructure:",squash"`

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	Networks []NetworkAttachment `hcl:"network,block" json:"networks,omitempty"` // Attach to the correct network // only when Image is specified

	// Tartget is the name of the container to attach to "conatiner.[name]"
	Target string `hcl:"target" json:"target"`

	// Ports to open, an ingress resource can be both a bridge between the host and the
	// an issolated network resource or bridge between two networks.
	// For this reason 3 different ports can be specified
	// Local - The docker network or container port to use for exposing the service, for example an ingress
	//         with the name consul-ingress configured to point to container.consul could set a local port 18500
	//         traffic on the same network would be able to reach the consul service on 8500 using the address
	//         consul-ingress.ingress.shipyard:18500
	// Remote - This is the destination port for the target container
	// Host   - The port to expose on localhost, this can be different from the Local container port.
	Ports []Port `hcl:"port,block" json:"ports,omitempty"`
}

// NewContainerIngress creates a new ingress for standard docker containers with the correct defaults
func NewContainerIngress(name string) *ContainerIngress {
	return &ContainerIngress{ResourceInfo: ResourceInfo{Name: name, Type: TypeContainerIngress, Status: PendingCreation}}
}
