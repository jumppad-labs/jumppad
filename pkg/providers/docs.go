package providers

import (
	"github.com/shipyard-run/cli/pkg/clients"
	"github.com/shipyard-run/cli/pkg/config"
)

// Docs defines a provider for creating documentation containers
type Docs struct {
	config *config.Docs
	client clients.Docker
}

// NewDocs creates a new Docs provider
func NewDocs(c *config.Docs, cc clients.Docker) *Docs {
	return &Docs{c, cc}
}

// Create a new documentation container
func (i *Docs) Create() error {
	// create the container config
	cc := &config.Container{}
	cc.Name = i.config.Name
	cc.NetworkRef = i.config.WANRef
	cc.Image = "shipyardrun/docs:latest"

	cc.Volumes = []config.Volume{
		config.Volume{
			Source:      i.config.Path + "/docs",
			Destination: "/app/docs",
		},
		config.Volume{
			Source:      i.config.Path + "/static",
			Destination: "/app/website/static",
		},
		config.Volume{
			Source:      i.config.Path + "/siteConfig.js",
			Destination: "/app/website/siteConfig.js",
		},
	}

	cc.Ports = []config.Port{
		config.Port{
			Protocol: "tcp",
			Host:     i.config.Port,
			Local:    3000,
		},
	}

	p := NewContainer(cc, i.client)

	return p.Create()
}

// Destroy the documentation container
func (i *Docs) Destroy() error {
	cc := &config.Container{
		Name:       i.config.Name,
		NetworkRef: i.config.WANRef,
	}

	p := NewContainer(cc, i.client)

	return p.Destroy()
}

// Lookup the ID of the documentation container
func (i *Docs) Lookup() (string, error) {
	cc := &config.Container{
		Name:       i.config.Name,
		NetworkRef: i.config.WANRef,
	}

	p := NewContainer(cc, i.client)

	return p.Lookup()
}
