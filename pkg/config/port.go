package config

// Port is a port mapping
type Port struct {
	Local    int    `hcl:"local"`             // Local port in the container
	Remote   int    `hcl:"remote"`            // Remote port of the service
	Host     int    `hcl:"host,optional"`     // Host port
	Protocol string `hcl:"protocol,optional"` // Protocol tcp, udp
}
