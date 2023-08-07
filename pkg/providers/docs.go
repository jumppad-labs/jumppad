package providers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/container"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

const docsImageName = "ghcr.io/jumppad-labs/docs"
const docsVersion = "v0.1.0"

type DocsConfig struct {
	DefaultPath string `json:"defaultPath"`
}

// Docs defines a provider for creating documentation containers
type Docs struct {
	config *resources.Docs
	client clients.ContainerTasks
	log    clients.Logger
}

// NewDocs creates a new Docs provider
func NewDocs(c *resources.Docs, cc clients.ContainerTasks, l clients.Logger) *Docs {
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
	ids, err := i.client.FindContainerIDs(i.config.FQRN)
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

func (d *Docs) Refresh() error {
	d.log.Debug("Refresh Docs", "ref", d.config.Name)

	configPath := utils.GetLibraryFolder("config", 0775)

	indices := []resources.IndexBook{}
	docsConfig := DocsConfig{}

	for index, book := range d.config.Content {
		br, err := d.config.ParentConfig.FindResource(book)
		if err != nil {
			return err
		}

		b := br.(*resources.Book)

		if index == 0 {
			if len(b.Index.Chapters) > 0 {
				if len(b.Index.Chapters[0].Pages) > 0 {
					docsConfig.DefaultPath = b.Index.Chapters[0].Pages[0].URI
				}
			}
		}

		indices = append(indices, b.Index)
	}

	docsConfigPath := filepath.Join(configPath, "jumppad.config.js")
	err := d.writeConfig(docsConfigPath, &docsConfig)
	if err != nil {
		return err
	}

	navigationPath := filepath.Join(configPath, "navigation.jsx")
	err = d.writeNavigation(navigationPath, indices)
	if err != nil {
		return err
	}

	progressPath := filepath.Join(configPath, "progress.jsx")
	err = d.writeProgress(progressPath)
	if err != nil {
		return err
	}

	return nil
}

func (c *Docs) Changed() (bool, error) {
	c.log.Debug("Checking changes", "ref", c.config.Name)

	return false, nil
}

func (d *Docs) createDocsContainer() error {
	// create the container config
	cc := &container.Container{
		ResourceMetadata: types.ResourceMetadata{
			Name:   d.config.Name,
			Type:   d.config.Type,
			Module: d.config.Module,
		},
	}
	cc.ParentConfig = d.config.Metadata().ParentConfig

	cc.Networks = d.config.Networks

	cc.Image = &resources.Image{Name: fmt.Sprintf("%s:%s", docsImageName, docsVersion)}

	// if image is set override defaults
	if d.config.Image != nil {
		cc.Image = d.config.Image
	}

	// pull the docker image
	err := d.client.PullImage(*cc.Image, false)
	if err != nil {
		return err
	}

	cc.Volumes = []resources.Volume{}

	// add the ports
	cc.Ports = []resources.Port{
		{
			Local:  "80",
			Remote: "80",
			Host:   fmt.Sprintf("%d", d.config.Port),
		},
	}

	// add the environment variables for the
	// ip and port of the terminal server
	localIP, _ := utils.GetLocalIPAndHostname()
	cc.Environment = map[string]string{
		"TERMINAL_SERVER_IP":   localIP,
		"TERMINAL_SERVER_PORT": "30003",
	}

	configPath := utils.GetLibraryFolder("config", 0775)
	contentPath := utils.GetLibraryFolder("content", 0775)

	indices := []resources.IndexBook{}
	docsConfig := DocsConfig{}

	for index, book := range d.config.Content {
		br, err := d.config.ParentConfig.FindResource(book)
		if err != nil {
			return err
		}

		b := br.(*resources.Book)

		bookPath := filepath.Join(contentPath, b.Name)
		destinationPath := fmt.Sprintf("/content/%s", b.Name)
		cc.Volumes = append(
			cc.Volumes,
			resources.Volume{
				Source:      bookPath,
				Destination: destinationPath,
			},
		)

		if index == 0 {
			if len(b.Index.Chapters) > 0 {
				if len(b.Index.Chapters[0].Pages) > 0 {
					docsConfig.DefaultPath = b.Index.Chapters[0].Pages[0].URI
				}
			}
		}

		indices = append(indices, b.Index)
	}

	docsConfigPath := filepath.Join(configPath, "jumppad.config.js")
	d.writeConfig(docsConfigPath, &docsConfig)

	docsConfigDestination := "/jumppad/jumppad.config.mjs"
	cc.Volumes = append(
		cc.Volumes,
		resources.Volume{
			Source:      docsConfigPath,
			Destination: docsConfigDestination,
		},
	)

	navigationPath := filepath.Join(configPath, "navigation.jsx")
	err = d.writeNavigation(navigationPath, indices)
	if err != nil {
		return err
	}

	navigationDestination := "/config/navigation.jsx"
	cc.Volumes = append(
		cc.Volumes,
		resources.Volume{
			Source:      navigationPath,
			Destination: navigationDestination,
		},
	)

	progressPath := filepath.Join(configPath, "progress.jsx")
	err = d.writeProgress(progressPath)
	if err != nil {
		return err
	}

	progressDestination := "/config/progress.jsx"
	cc.Volumes = append(
		cc.Volumes,
		resources.Volume{
			Source:      progressPath,
			Destination: progressDestination,
		},
	)

	// set the FQDN
	fqdn := utils.FQDN(d.config.Name, d.config.Module, d.config.Type)
	d.config.FQRN = fqdn

	_, err = d.client.CreateContainer(cc)
	return err
}

func (i *Docs) writeConfig(configPath string, config *DocsConfig) error {
	configJSON, err := json.MarshalIndent(config, "", " ")
	if err != nil {
		return err
	}

	configJS := fmt.Sprintf("export const jumppad = %s", configJSON)
	err = os.WriteFile(configPath, []byte(configJS), 0755)
	if err != nil {
		return fmt.Errorf("Unable to write config to disk at %s", configPath)
	}

	return nil
}

func (i *Docs) writeNavigation(navigationPath string, indices []resources.IndexBook) error {
	navigationJSON, err := json.MarshalIndent(indices, "", " ")
	if err != nil {
		return err
	}

	navigationJSX := fmt.Sprintf("export const navigation = %s", navigationJSON)
	err = os.WriteFile(navigationPath, []byte(navigationJSX), 0755)
	if err != nil {
		return fmt.Errorf("Unable to write navigation to disk at %s", navigationPath)
	}

	return nil
}

// Add optional task progress
func (i *Docs) writeProgress(progressPath string) error {
	tasks, _ := i.config.ParentConfig.FindResourcesByType(resources.TypeTask)

	progress := []resources.Progress{}
	for _, tr := range tasks {
		task := tr.(*resources.Task)
		progress = append(progress, task.Progress)
	}

	progressJSON, err := json.MarshalIndent(progress, "", " ")
	if err != nil {
		return err
	}

	progressJSX := fmt.Sprintf("export const progress = %s", progressJSON)
	err = os.WriteFile(progressPath, []byte(progressJSX), 0755)
	if err != nil {
		return fmt.Errorf("Unable to write progress to disk at %s", progressPath)
	}

	return nil
}
