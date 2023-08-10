package docs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/container"
	"github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

const docsImageName = "ghcr.io/jumppad-labs/docs"
const docsVersion = "v0.1.0"

type DocsConfig struct {
	DefaultPath string `json:"defaultPath"`
	Logo        Logo   `json:"logo"`
}

type Progress struct {
	ID            string              `json:"id"`
	Prerequisites []string            `json:"prerequisites"`
	Conditions    []ProgressCondition `json:"conditions"`
	Status        string              `json:"status"`
}

type ProgressCondition struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// Docs defines a provider for creating documentation containers
type DocsProvider struct {
	config *Docs
	client container.ContainerTasks
	log    logger.Logger
}

func (p *DocsProvider) Init(cfg htypes.Resource, l logger.Logger) error {
	c, ok := cfg.(*Docs)
	if !ok {
		return fmt.Errorf("unable to initialize Docs provider, resource is not of type Docs")
	}

	cli, err := clients.GenerateClients(l)
	if err != nil {
		return err
	}

	p.config = c
	p.client = cli.ContainerTasks
	p.log = l

	return nil
}

// Create a new documentation container
func (p *DocsProvider) Create() error {
	p.log.Info("Creating Documentation", "ref", p.config.ID)

	// create the documentation container
	err := p.createDocsContainer()
	if err != nil {
		return err
	}

	return nil
}

// Destroy the documentation container
func (p *DocsProvider) Destroy() error {
	p.log.Info("Destroy Documentation", "ref", p.config.ID)

	// remove the docs
	ids, err := p.client.FindContainerIDs(p.config.ContainerName)
	if err != nil {
		return err
	}

	for _, id := range ids {
		err := p.client.RemoveContainer(id, true)
		if err != nil {
			return err
		}
	}

	return nil
}

// Lookup the ID of the documentation container
func (p *DocsProvider) Lookup() ([]string, error) {
	return p.client.FindContainerIDs(p.config.ContainerName)
}

func (p *DocsProvider) Refresh() error {
	p.log.Debug("Refresh Docs", "ref", p.config.ID)

	configPath := utils.GetLibraryFolder("config", 0775)

	indices := []Index{}
	docsConfig := DocsConfig{
		Logo: p.config.Logo,
	}

	for index, book := range p.config.Content {
		if index == 0 {
			if len(book.Index.Chapters) > 0 {
				if len(book.Index.Chapters[0].Pages) > 0 {
					docsConfig.DefaultPath = book.Index.Chapters[0].Pages[0].URI
				}
			}
		}

		indices = append(indices, book.Index)
	}

	docsConfigPath := filepath.Join(configPath, "jumppad.config.js")
	err := p.writeConfig(docsConfigPath, &docsConfig)
	if err != nil {
		return err
	}

	navigationPath := filepath.Join(configPath, "navigation.jsx")
	err = p.writeNavigation(navigationPath, indices)
	if err != nil {
		return err
	}

	progressPath := filepath.Join(configPath, "progress.jsx")
	err = p.writeProgress(progressPath)
	if err != nil {
		return err
	}

	return nil
}

func (p *DocsProvider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.ID)

	return false, nil
}

func (p *DocsProvider) createDocsContainer() error {
	// set the FQDN
	fqdn := utils.FQDN(p.config.Name, p.config.Module, p.config.Type)
	p.config.ContainerName = fqdn

	// create the container config
	cc := &types.Container{
		Name: fqdn,
	}

	cc.Networks = p.config.Networks.ToClientNetworkAttachments()

	cc.Image = &types.Image{Name: fmt.Sprintf("%s:%s", docsImageName, docsVersion)}

	// if image is set override defaults
	if p.config.Image != nil {
		cc.Image = &types.Image{
			Name:     p.config.Image.Name,
			Username: p.config.Image.Username,
			Password: p.config.Image.Password,
		}
	}

	// pull the docker image
	err := p.client.PullImage(*cc.Image, false)
	if err != nil {
		return err
	}

	// add the ports
	cc.Ports = []types.Port{
		{
			Local:  "80",
			Remote: "80",
			Host:   fmt.Sprintf("%d", p.config.Port),
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

	indices := []Index{}
	docsConfig := DocsConfig{
		Logo: p.config.Logo,
	}

	for i, book := range p.config.Content {
		bookPath := filepath.Join(contentPath, book.Name)
		destinationPath := fmt.Sprintf("/content/%s", book.Name)
		cc.Volumes = append(
			cc.Volumes,
			types.Volume{
				Source:      bookPath,
				Destination: destinationPath,
			},
		)

		if i == 0 {
			if len(book.Index.Chapters) > 0 {
				if len(book.Index.Chapters[0].Pages) > 0 {
					docsConfig.DefaultPath = book.Index.Chapters[0].Pages[0].URI
				}
			}
		}

		indices = append(indices, book.Index)
	}

	docsConfigPath := filepath.Join(configPath, "jumppad.config.js")
	p.writeConfig(docsConfigPath, &docsConfig)

	cc.Volumes = append(
		cc.Volumes,
		types.Volume{
			Source:      docsConfigPath,
			Destination: "/jumppad/jumppad.config.mjs",
		},
	)

	assetsDestination := "/jumppad/public/assets"
	cc.Volumes = append(
		cc.Volumes,
		types.Volume{
			Source:      p.config.Assets,
			Destination: assetsDestination,
		},
	)

	navigationPath := filepath.Join(configPath, "navigation.jsx")
	err = p.writeNavigation(navigationPath, indices)
	if err != nil {
		return err
	}

	cc.Volumes = append(
		cc.Volumes,
		types.Volume{
			Source:      navigationPath,
			Destination: "/config/navigation.jsx",
		},
	)

	progressPath := filepath.Join(configPath, "progress.jsx")
	err = p.writeProgress(progressPath)
	if err != nil {
		return err
	}

	cc.Volumes = append(
		cc.Volumes,
		types.Volume{
			Source:      progressPath,
			Destination: "/config/progress.jsx",
		},
	)

	_, err = p.client.CreateContainer(cc)
	return err
}

func (p *DocsProvider) writeConfig(configPath string, config *DocsConfig) error {
	configJSON, err := json.MarshalIndent(config, "", " ")
	if err != nil {
		return err
	}

	configJS := fmt.Sprintf("export const jumppad = %s", configJSON)
	err = os.WriteFile(configPath, []byte(configJS), 0755)
	if err != nil {
		return fmt.Errorf("unable to write config to disk at %s", configPath)
	}

	return nil
}

func (p *DocsProvider) writeNavigation(navigationPath string, indices []Index) error {
	navigationJSON, err := json.MarshalIndent(indices, "", " ")
	if err != nil {
		return err
	}

	navigationJSX := fmt.Sprintf("export const navigation = %s", navigationJSON)
	err = os.WriteFile(navigationPath, []byte(navigationJSX), 0755)
	if err != nil {
		return fmt.Errorf("unable to write navigation to disk at %s", navigationPath)
	}

	return nil
}

// Add optional task progress
func (p *DocsProvider) writeProgress(progressPath string) error {
	progress := []Progress{}
	for _, book := range p.config.Content {
		for _, chapter := range book.Chapters {
			for _, task := range chapter.Tasks {
				p := Progress{
					ID:            task.ID,
					Prerequisites: task.Prerequisites,
					Status:        "locked",
				}

				if len(task.Prerequisites) == 0 {
					p.Status = "unlocked"
				}

				for _, condition := range task.Conditions {
					p.Conditions = append(p.Conditions, ProgressCondition{
						ID:          condition.Name,
						Description: condition.Description,
						Status:      "",
					})
				}

				progress = append(progress, p)
			}
		}
	}

	progressJSON, err := json.MarshalIndent(progress, "", " ")
	if err != nil {
		return err
	}

	progressJSX := fmt.Sprintf("export const progress = %s", progressJSON)
	err = os.WriteFile(progressPath, []byte(progressJSX), 0755)
	if err != nil {
		return fmt.Errorf("unable to write progress to disk at %s", progressPath)
	}

	return nil
}
