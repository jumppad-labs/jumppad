package config

// Network defines a Docker network
type Network struct {
	Name   string
	Subnet string `hcl:"subnet"`
}
