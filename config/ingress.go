package config

type Ingress struct {
	Name      string
	targetRef interface{}

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
