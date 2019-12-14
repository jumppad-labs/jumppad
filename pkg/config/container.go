package config

// Container defines a structure for creating Docker containers
type Container struct {
	Name        string
	Image       string   `hcl:"image"`
	Command     []string `hcl:"command,optional"`
	Environment []KV     `hcl:"env,block"`
	Volumes     []Volume `hcl:"volume,block"`
	Network     string   `hcl:"network"`
	NetworkRef  *Network
	WANRef      *Network
	IPAddress   string `hcl:"ip_address,optional"`
	Ports       []Port `hcl:"port,block"`
	Privileged  bool   `hcl:"privileged,optional"`
}

// Volume defines a Docker Volume
type Volume struct {
	Source      string `hcl:"source"`
	Destination string `hcl:"destination"`
}

// KV is a key/value type
type KV struct {
	Key   string `hcl:"key"`
	Value string `hcl:"value"`
}
