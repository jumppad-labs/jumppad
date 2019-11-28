package config

// Network defines a Docker network
type Network struct {
	name   string
	Subnet string `hcl:"subnet"`
}
