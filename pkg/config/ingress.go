package config

// Ingress defines an ingress service mapping ports between local host or docker network and the target
type Ingress struct {
	Name       string
	TargetRef  interface{}
	NetworkRef *Network // automatically fetched from target
	WANRef     *Network // automatically created

	Target    string `hcl:"target"`
	Service   string `hcl:"service,optional"`
	Namespace string `hcl:"namespace,optional"`
	Ports     []Port `hcl:"port,block"`
	IPAddress string `hcl:"ip_address,optional"`
}
