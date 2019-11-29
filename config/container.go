package config

type Container struct {
	Name       string
	Image      string   `hcl:"image"`
	Command    []string `hcl:"command,optional"`
	Volumes    []Volume `hcl:"volume,block"`
	Network    string   `hcl:"network"`
	networkRef *Network
	wanRef     *Network
	IPAddress  string `hcl:"ip_address,optional"`
}

type Volume struct {
	Source      string `hcl:"source"`
	Destination string `hcl:"destination"`
}
