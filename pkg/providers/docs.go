package providers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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

	indices := []resources.IndexBook{}

	contentPath := utils.GetLibraryFolder("content", 0775)
	configPath := utils.GetLibraryFolder("config", 0775)
	checksPath := utils.GetLibraryFolder("checks", 0775)

	// Add book content and navigation
	for _, book := range i.config.Content {
		br, err := i.config.ParentConfig.FindResource(book)
		if err != nil {
			return err
		}

		b := br.(*resources.Book)

		sourcePath := filepath.Join(contentPath, b.Name)
		destinationPath := filepath.Join("/content", b.Name)
		cc.Volumes = append(
			cc.Volumes,
			resources.Volume{
				Source:      sourcePath,
				Destination: destinationPath,
			},
		)

		indices = append(indices, b.Index)
	}

	navigationJSON, err := json.MarshalIndent(indices, "", " ")
	if err != nil {
		return err
	}

	navigationJSX := fmt.Sprintf("export const navigation = %s", navigationJSON)
	navigationSource := filepath.Join(configPath, "navigation.jsx")
	err = os.WriteFile(navigationSource, []byte(navigationJSX), 0755)
	if err != nil {
		return fmt.Errorf("Unable to write navigation to disk at %s", navigationSource)
	}

	navigationDestination := filepath.Join("/config", "navigation.jsx")
	cc.Volumes = append(
		cc.Volumes,
		resources.Volume{
			Source:      navigationSource,
			Destination: navigationDestination,
		},
	)

	// Add task progress
	tasks, err := i.config.ParentConfig.FindResourcesByType(resources.TypeTask)
	if err != nil {
		return err
	}

	progress := []resources.Progress{}
	checks := []resources.Validation{}

	for _, tr := range tasks {
		task := tr.(*resources.Task)

		progress = append(progress, task.Progress)
		checks = append(checks, task.Validation)
	}

	progressJSON, err := json.MarshalIndent(progress, "", " ")
	if err != nil {
		return err
	}

	progressJSX := fmt.Sprintf("export const progress = %s", progressJSON)
	progressSource := filepath.Join(configPath, "progress.jsx")
	err = os.WriteFile(progressSource, []byte(progressJSX), 0755)
	if err != nil {
		return fmt.Errorf("Unable to write progress to disk at %s", progressSource)
	}

	progressDestination := filepath.Join("/config", "progress.jsx")
	cc.Volumes = append(
		cc.Volumes,
		resources.Volume{
			Source:      progressSource,
			Destination: progressDestination,
		},
	)

	checksJSON, err := json.MarshalIndent(checks, "", " ")
	if err != nil {
		return err
	}

	checksSource := filepath.Join(checksPath, "checks.json")
	err = os.WriteFile(checksSource, []byte(checksJSON), 0755)
	if err != nil {
		return fmt.Errorf("Unable to write checks configuration to disk at %s", checksSource)
	}

	// cc.Volumes = append(
	// 	cc.Volumes,
	// 	resources.Volume{
	// 		Source:      checksPath,
	// 		Destination: "/checks",
	// 	},
	// )

	// add the ports
	cc.Ports = []resources.Port{
		{
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
