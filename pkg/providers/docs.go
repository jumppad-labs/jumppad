package providers

import (
	"fmt"
	"html/template"
	"io/ioutil"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"golang.org/x/xerrors"
)

const docsImageName = "shipyardrun/docs"
const docsVersion = "v0.0.6"

const terminalImageName = "shipyardrun/terminal-server"
const terminalVersion = "latest"

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

	// set the state
	i.config.Status = config.Applied

	return nil
}

func (i *Docs) createDocsContainer() error {
	// create the container config
	cc := config.NewContainer(i.config.Name)
	i.config.ResourceInfo.AddChild(cc)

	cc.Networks = i.config.Networks

	cc.Image = config.Image{Name: fmt.Sprintf("%s:%s", docsImageName, docsVersion)}

	// if image is set override defaults
	if i.config.Image != nil {
		cc.Image = *i.config.Image
	}

	// pull the docker image
	err := i.client.PullImage(cc.Image, false)
	if err != nil {
		return err
	}

	cc.Volumes = []config.Volume{}

	if i.config.Path != "" {
		cc.Volumes = append(
			cc.Volumes,
			config.Volume{
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
			config.Volume{
				Source:      indexPath,
				Destination: "/shipyard/sidebars.js",
			},
		)
	}

	// add the ports
	cc.Ports = []config.Port{
		// set the doumentation port
		config.Port{
			Local:  "80",
			Remote: "80",
			Host:   fmt.Sprintf("%d", i.config.Port),
		},
		// set the livereload port
		config.Port{
			Local:  "37950",
			Remote: "37950",
			Host:   "37950",
		},
	}

	_, err = i.client.CreateContainer(cc)
	return err
}

func (i *Docs) createTerminalContainer() error {
	// create the container config
	cc := config.NewContainer("terminal")
	i.config.ResourceInfo.AddChild(cc)

	cc.Networks = i.config.Networks
	cc.Image = config.Image{Name: fmt.Sprintf("%s:%s", terminalImageName, terminalVersion)}

	// pull the image
	err := i.client.PullImage(cc.Image, false)
	if err != nil {
		return err
	}

	// TODO we are mounting the docker sock, need to look at how this works on Windows
	cc.Volumes = make([]config.Volume, 0)
	cc.Volumes = append(
		cc.Volumes,
		config.Volume{
			Source:      utils.GetDockerSock(),
			Destination: utils.GetDockerSock(),
		},
	)

	cc.Ports = []config.Port{
		config.Port{
			Protocol: "tcp",
			Host:     "27950",
			Local:    "27950",
		},
	}

	_, err = i.client.CreateContainer(cc)
	return err
}

// Destroy the documentation container
func (i *Docs) Destroy() error {
	i.log.Info("Destroy Documentation", "ref", i.config.Name)

	// remove the docs
	ids, err := i.client.FindContainerIDs(i.config.Name, i.config.Type)
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
	ids, err = i.client.FindContainerIDs("terminal", i.config.Type)
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
