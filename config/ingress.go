package config

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

type Port struct {
	Local  int `hcl:"local"`
	Remote int `hcl:"remote"`
	Host   int `hcl:"host,optional"`
}
