package config

type Ingress struct {
	name      string
	targetRef interface{}

	Target  string `hcl:"target"`
	Service string `hcl:"service,optional"`
	Ports   []Port `hcl:"port,block"`
}

type Port struct {
	Local  int `hcl:"local"`
	Remote int `hcl:"remote"`
	Host   int `hcl:"host,optional"`
}
