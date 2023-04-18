package providers

import (
	"fmt"
	"html/template"
	"io/ioutil"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/hclconfig/types"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config/resources"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"golang.org/x/xerrors"
)

const docsImageName = "shipyardrun/docs"
const docsVersion = "v0.6.2"

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

	// set the default live reload port
	if i.config.LiveReloadPort == 0 {
		i.config.LiveReloadPort = 37950
	}

	// create the documentation container
	err := i.createDocsContainer()
	if err != nil {
		return err
	}

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
				Destination: "/shipyard/docs",
			},
		)
	}

	// if the index pages have been set
	// generate the javascript
	if i.config.IndexTitle != "" && len(i.config.IndexPages) > 0 {
		indexPath, err := i.generateDocusaursIndex(i.config.IndexTitle, i.config.IndexPages)
		if err != nil {
			return xerrors.Errorf("Unable to generate index for documentation: %w", err)
		}

		cc.Volumes = append(
			cc.Volumes,
			resources.Volume{
				Source:      indexPath,
				Destination: "/shipyard/sidebars.js",
			},
		)
	}

	// add the ports
	cc.Ports = []resources.Port{
		// set the doumentation port
		resources.Port{
			Local:  "80",
			Remote: "80",
			Host:   fmt.Sprintf("%d", i.config.Port),
		},
		// set the livereload port
		resources.Port{
			Local:  "37950",
			Remote: "37950",
			Host:   fmt.Sprintf("%d", i.config.LiveReloadPort),
		},
	}

	// add the environment variables for the
	// ip and port of the terminal server
	localIP, _ := utils.GetLocalIPAndHostname()
	cc.Env = map[string]string{
		"TERMINAL_SERVER_IP":   localIP,
		"TERMINAL_SERVER_PORT": "30003",
	}

	_, err = i.client.CreateContainer(cc)
	return err
}

// Destroy the documentation container
func (i *Docs) Destroy() error {
	i.log.Info("Destroy Documentation", "ref", i.config.Name)

	// remove the docs
	ids, err := i.client.FindContainerIDs(i.config.ID)
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

func (i *Docs) generateDocusaursIndex(title string, pages []string) (string, error) {
	tmpFile, err := ioutil.TempFile(utils.ShipyardTemp(), "*.json")
	if err != nil {
		return "", err
	}

	data := struct {
		Title string
		Pages []string
	}{
		title,
		pages,
	}

	t := template.Must(template.New("pages").Parse(sideBarsTemplate))
	err = t.Execute(tmpFile, data)
	if err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}

var sideBarsTemplate = `
module.exports = {
    docs: {
      {{.Title}}: [
		{{- $first := true -}}
		{{- range .Pages -}}
	 		{{- if $first -}}
        		{{- $first = false -}}
    		{{- else -}}
        		,
			{{- end}}
			"{{- .}}"
		{{- end}}	
	  ]
    },
  }
`
