package providers

import (
	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
)

// Docs defines a provider for creating documentation containers
type Docs struct {
	config *config.Docs
	client clients.Docker
	log    hclog.Logger
}

// NewDocs creates a new Docs provider
func NewDocs(c *config.Docs, cc clients.Docker, l hclog.Logger) *Docs {
	return &Docs{c, cc, l}
}

// Create a new documentation container
func (i *Docs) Create() error {
	i.log.Info("Creating Documentation", "ref", i.config.Name)

	// create the documentation container
	err := i.createDocsContainer()
	if err != nil {
		return err
	}

	// create the terminal server container
	err = i.createTerminalContainer()
	if err != nil {
		return err
	}

	return nil
}

func (i *Docs) createDocsContainer() error {
	/*
		// create the container config
		cc := &config.Container{}
		cc.Name = i.config.Name
		cc.NetworkRef = i.config.WANRef
		cc.Image = config.Image{Name: "shipyardrun/docs:latest"}

		cc.Volumes = []config.Volume{}

		if i.config.Path != "" {
			cc.Volumes = append(
				cc.Volumes,
				config.Volume{
					Source:      i.config.Path + "/docs",
					Destination: "/shipyard/docs",
				},
			)

			siteConfigPath := filepath.Join(i.config.Path, "siteConfig.js")
			_, err := os.Stat(siteConfigPath)
			if err == nil {
				cc.Volumes = append(
					cc.Volumes,
					config.Volume{
						Source:      i.config.Path + "/siteConfig.js",
						Destination: "/shipyard/siteConfig.js",
					},
				)
			}

			sidebarsPath := filepath.Join(i.config.Path, "sidebars.js")
			_, err = os.Stat(sidebarsPath)
			if err == nil {
				cc.Volumes = append(
					cc.Volumes,
					config.Volume{
						Source:      i.config.Path + "/sidebars.js",
						Destination: "/shipyard/sidebars.js",
					},
				)
			}
		}

		/*
			config.Volume{
				Source:      i.config.Path + "/static",
				Destination: "/shipyard/website/static",
			},
			config.Volume{
				Source:      i.config.Path + "/siteConfig.js",
				Destination: "/shipyard/website/siteConfig.js",
			},

		cc.Ports = []config.Port{
			config.Port{
				Protocol: "tcp",
				Host:     i.config.Port,
				Local:    3000,
			},
		}

		p := NewContainer(cc, i.client, i.log.With("parent_ref", i.config.Name))
	*/

	return nil
}

func (i *Docs) createTerminalContainer() error {
	/*
		// create the container config
		cc := &config.Container{}
		cc.Name = "terminal"
		cc.NetworkRef = i.config.WANRef
		cc.Image = config.Image{Name: "shipyardrun/terminal-server:latest"}

		// TODO we are mounting the docker sock, need to look at how this works on Windows
		cc.Volumes = make([]config.Volume,0)
		cc.Volumes = append(
			cc.Volumes,
			config.Volume{
				Source:      "/var/run/docker.sock",
				Destination: "/var/run/docker.sock",
			},
		)

		cc.Ports = []config.Port{
			config.Port{
				Protocol: "tcp",
				Host:     27950,
				Local:    27950,
			},
		}

		p := NewContainer(cc, i.client, i.log.With("parent_ref", i.config.Name))

		return p.Create()
	*/

	return nil
}

// Destroy the documentation container
func (i *Docs) Destroy() error {
	/*
		i.log.Info("Destroy Documentation", "ref", i.config.Name)

		cc := &config.Container{
			Name:       i.config.Name,
			NetworkRef: i.config.WANRef,
		}

		p := NewContainer(cc, i.client, i.log.With("parent_ref", i.config.Name))
		err := p.Destroy()
		if err != nil {
			return err
		}

		cc = &config.Container{
			Name:       "terminal",
			NetworkRef: i.config.WANRef,
		}

		p = NewContainer(cc, i.client, i.log.With("parent_ref", i.config.Name))
		err = p.Destroy()
		if err != nil {
			return err
		}
	*/

	return nil
}

// Lookup the ID of the documentation container
func (i *Docs) Lookup() (string, error) {
	/*
		cc := &config.Container{
			Name:       i.config.Name,
			NetworkRef: i.config.WANRef,
		}

		p := NewContainer(cc, i.client, i.log.With("parent_ref", i.config.Name))
	*/

	return "", nil
}
