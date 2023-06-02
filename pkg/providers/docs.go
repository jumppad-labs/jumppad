package providers

import (
	"fmt"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

const docsImageName = "ghcr.io/jumppad-labs/docs"
const docsVersion = "dev"

// Docs defines a provider for creating documentation containers
type Docs struct {
	config *resources.Docs
	client clients.ContainerTasks
	log    hclog.Logger
}

// NewDocs creates a new Docs provider
func NewDocs(c *resources.Docs, cc clients.ContainerTasks, l hclog.Logger) *Docs {
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

	return nil
}

// Destroy the documentation container
func (i *Docs) Destroy() error {
	i.log.Info("Destroy Documentation", "ref", i.config.Name)

	// remove the docs
	ids, err := i.client.FindContainerIDs(i.config.FQDN)
	if err != nil {
		return err
	}

	for _, id := range ids {
		err := i.client.RemoveContainer(id, true)
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

func (c *Docs) Refresh() error {
	c.log.Info("Refresh Docs", "ref", c.config.Name)

	return nil
}

func (i *Docs) createDocsContainer() error {
	// create the container config
	cc := &resources.Container{
		ResourceMetadata: types.ResourceMetadata{
			Name:   i.config.Name,
			Type:   i.config.Type,
			Module: i.config.Module,
		},
	}
	cc.ParentConfig = i.config.Metadata().ParentConfig

	cc.Networks = i.config.Networks

	cc.Image = &resources.Image{Name: fmt.Sprintf("%s:%s", docsImageName, docsVersion)}

	// if image is set override defaults
	if i.config.Image != nil {
		cc.Image = i.config.Image
	}

	// pull the docker image
	err := i.client.PullImage(*cc.Image, false)
	if err != nil {
		return err
	}

	cc.Volumes = []resources.Volume{}

	if i.config.Path != "" {
		cc.Volumes = append(
			cc.Volumes,
			resources.Volume{
				Source:      i.config.Path,
				Destination: "/content",
			},
		)
		cc.Volumes = append(
			cc.Volumes,
			resources.Volume{
				Source:      i.config.NavigationFile,
				Destination: "/config/navigation.jsx",
			},
		)
	}

	// add the ports
	cc.Ports = []resources.Port{
		resources.Port{
			Local:  "80",
			Remote: "80",
			Host:   fmt.Sprintf("%d", i.config.Port),
		},
	}

	// add the environment variables for the
	// ip and port of the terminal server
	localIP, _ := utils.GetLocalIPAndHostname()
	cc.Environment = map[string]string{
		"TERMINAL_SERVER_IP":   localIP,
		"TERMINAL_SERVER_PORT": "30003",
	}

	// set the FQDN
	fqdn := utils.FQDN(i.config.Name, i.config.Module, i.config.Type)
	i.config.FQDN = fqdn

	_, err = i.client.CreateContainer(cc)
	return err
}
