package config

// Config defines the stack config
type Config struct {
	Blueprint   *Blueprint
	WAN         *Network
	Docs        *Docs
	Clusters    []*Cluster
	Containers  []*Container
	Networks    []*Network
	HelmCharts  []*Helm
	K8sConfig   []*K8sConfig
	Ingresses   []*Ingress
	LocalExecs  []*LocalExec
	RemoteExecs []*RemoteExec
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

// ResourceCount defines the number of resources in a config
func (c *Config) ResourceCount() int {
	// start at 1 as we always have a wan
	co := 1

	if c.Docs != nil {
		co++
	}

	co += len(c.Clusters)
	co += len(c.Containers)
	co += len(c.Containers)
	co += len(c.HelmCharts)
	co += len(c.K8sConfig)
	co += len(c.Ingresses)
	co += len(c.LocalExecs)
	co += len(c.RemoteExecs)

	return co
}
