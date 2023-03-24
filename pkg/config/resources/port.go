package resources

// Port is a port mapping
type Port struct {
	Local         string `hcl:"local" json:"local"`                                                             // Local port in the container
	Remote        string `hcl:"remote" json:"remote"`                                                           // Remote port of the service
	Host          string `hcl:"host,optional" json:"host,omitempty"`                                            // Host port
	Protocol      string `hcl:"protocol,optional" json:"protocol,omitempty"`                                    // Protocol tcp, udp
	OpenInBrowser string `hcl:"open_in_browser,optional" json:"open_in_browser" mapstructure:"open_in_browser"` // When a host port is defined open this port with the given path in a browser
}

// PortRange allows a range of ports to be mapped
type PortRange struct {
	Range      string `hcl:"range" json:"local" mapstructure:"local"`                                      // Local port in the container
	EnableHost bool   `hcl:"enable_host,optional" json:"enable_host,omitempty" mapstructure:"enable_host"` // Host port
	Protocol   string `hcl:"protocol,optional" json:"protocol,omitempty"`                                  // Protocol tcp, udp
}
