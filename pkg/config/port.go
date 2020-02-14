package config

// Port is a port mapping
type Port struct {
	Local    int    `hcl:"local" json:"local"`                          // Local port in the container
	Remote   int    `hcl:"remote" json:"remote"`                        // Remote port of the service
	Host     int    `hcl:"host,optional" json:"host,omitempty"`         // Host port
	Protocol string `hcl:"protocol,optional" json:"protocol,omitempty"` // Protocol tcp, udp
}
