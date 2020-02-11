package config

// Network defines a Docker network
type Network struct {
	Name   string
	State  State
	Subnet string `hcl:"subnet"`
}
