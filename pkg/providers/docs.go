package providers

import (
	"os"
	"path/filepath"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
)

// Docs defines a provider for creating documentation containers
type Docs struct {
	config *config.Docs
	client clients.ContainerTasks
	log    hclog.Logger
}

// NewDocs creates a new Docs provider
func NewDocs(c *config.Docs, cc clients.ContainerTasks, l hclog.Logger) *Docs {
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
	// create the container config
	cc := config.Container{}
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

	// add the ports
	cc.Ports = []config.Port{
		// set the doumentation port
		config.Port{
			Local:  3000,
			Remote: 3000,
			Host:   i.config.Port,
		},
		// set the livereload port
		config.Port{
			Local:  37950,
			Remote: 37950,
			Host:   37950,
		},
	}

	_, err := i.client.CreateContainer(cc)
	return err
}

func (i *Docs) createTerminalContainer() error {
	// create the container config
	cc := config.Container{}
	cc.Name = "terminal"
	cc.NetworkRef = i.config.WANRef
	cc.Image = config.Image{Name: "shipyardrun/terminal-server:latest"}

	// TODO we are mounting the docker sock, need to look at how this works on Windows
	cc.Volumes = make([]config.Volume, 0)
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

	_, err := i.client.CreateContainer(cc)
	return err
}

// Destroy the documentation container
func (i *Docs) Destroy() error {
	i.log.Info("Destroy Documentation", "ref", i.config.Name)

	// remove the docs
	ids, err := i.client.FindContainerIDs(i.config.Name, i.config.WANRef.Name)
	if err != nil {
		return err
	}

	for _, id := range ids {
		err := i.client.RemoveContainer(id)
		if err != nil {
			return err
		}
	}

	// remove the terminal server
	ids, err = i.client.FindContainerIDs("terminal", i.config.WANRef.Name)
	for _, id := range ids {
		err := i.client.RemoveContainer(id)
		if err != nil {
			return err
		}
	}
	return nil
}

// Lookup the ID of the documentation container
func (i *Docs) Lookup() ([]string, error) {
	/*
		cc := &config.Container{
			Name:       i.config.Name,
			NetworkRef: i.config.WANRef,
		}

		p := NewContainer(cc, i.client, i.log.With("parent_ref", i.config.Name))
	*/

	return []string{}, nil
}
