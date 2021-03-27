package config

// TypeIngress is the resource string for the type
const TypeLegacyIngress ResourceType = "legacy_ingress"

// Ingress defines an ingress service mapping ports between local host or docker network and the target
// Note: This type is Deprecated and will be removed in a later version
//       Please use one of the new specific types:
//       * K8sIngress
//       * NomadIngress
//       * ContainerIngress
type LegacyIngress struct {
	ResourceInfo `hcl:",remain" mapstructure:",squash"`

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	Networks []NetworkAttachment `hcl:"network,block" json:"networks,omitempty"` // Attach to the correct network // only when Image is specified

	Target    string `hcl:"target" json:"target"`
	Service   string `hcl:"service,optional" json:"service,omitempty"`
	Namespace string `hcl:"namespace,optional" json:"namespace,omitempty"`
	Ports     []Port `hcl:"port,block" json:"ports,omitempty"`
}

// NewIngress creates a new ingress with the correct defaults
func NewLegacyIngress(name string) *LegacyIngress {
	return &LegacyIngress{ResourceInfo: ResourceInfo{Name: name, Type: TypeLegacyIngress, Status: PendingCreation}}
}
