package config

// Port is a port mapping
type Port struct {
	Local         string `hcl:"local" json:"local"`                                                             // Local port in the container
	Remote        string `hcl:"remote" json:"remote"`                                                           // Remote port of the service
	Host          string `hcl:"host,optional" json:"host,omitempty"`                                            // Host port
	Protocol      string `hcl:"protocol,optional" json:"protocol,omitempty"`                                    // Protocol tcp, udp
	OpenInBrowser bool   `hcl:"open_in_browser,optional" json:"open_in_browser" mapstructure:"open_in_browser"` // When a host port is defined open the location in a browser
}
