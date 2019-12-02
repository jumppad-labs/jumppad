package config

// Ingress defines an ingress service mapping ports between local host or docker network and the target
type Ingress struct {
	Name       string
	targetRef  interface{}
	networkRef *Network // automatically fetched from target
	wanRef     *Network // automatically created

	Target    string `hcl:"target"`
	Service   string `hcl:"service,optional"`
	Ports     []Port `hcl:"port,block"`
	IPAddress string `hcl:"ip_address,optional"`
}
