package config

// Config defines the stack config
type Config struct {
	Blueprint  *Blueprint
	WAN        *Network
	Docs       *Docs
	Clusters   []*Cluster
	Containers []*Container
	Networks   []*Network
	HelmCharts []*Helm
	Ingresses  []*Ingress
}

// New creates a new Config with the default WAN network
func New() (*Config, error) {
	c := &Config{}

	// add the default WAN
	c.WAN = &Network{
		Name:   "wan",
		Subnet: "10.200.0.0/16",
	}

	// TODO load wan settings from defaults

	return c, nil
}
