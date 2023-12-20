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
	"github.com/jumppad-labs/jumppad/pkg/utils"
	sdk "github.com/jumppad-labs/plugin-sdk"
)

const docsImageName = "ghcr.io/jumppad-labs/docs"
const docsVersion = "v0.4.0"

type DocsConfig struct {
	DefaultPath string `json:"defaultPath"`
	Logo        Logo   `json:"logo"`
}

type State struct {
	Index struct {
		Title    string `json:"title"`
		Chapters []struct {
			Title string `json:"title,omitempty"`
			Pages []struct {
				Title string `json:"title"`
				URI   string `json:"uri"`
			} `json:"pages"`
		} `json:"chapters"`
	} `json:"index"`

	Progress []struct {
		ID            string   `json:"id"`
		Prerequisites []string `json:"prerequisites"`
		Conditions    []struct {
			ID          string `json:"id"`
			Description string `json:"description"`
			Status      string `json:"status"`
		} `json:"conditions"`
		Status string `json:"status"`
	} `json:"progress"`
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
	log    sdk.Logger
}

func (p *DocsProvider) Init(cfg htypes.Resource, l sdk.Logger) error {
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

	// write the content
	return p.Refresh()
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

	// remove the cached files
	contentPath := utils.LibraryFolder("", 0775)
	os.RemoveAll(contentPath)

	return nil
}

// Lookup the ID of the documentation container
func (p *DocsProvider) Lookup() ([]string, error) {
	return p.client.FindContainerIDs(p.config.ContainerName)
}

func (p *DocsProvider) Refresh() error {
	changed, err := p.checkChanged()
	if err != nil {
		return fmt.Errorf("unable to check if content has changed: %s", err)
	}

	// no changes return
	if !changed {
		return nil
	}

	p.log.Info("Refresh Docs", "ref", p.config.ID)

	// refresh content on disk
	configPath := utils.LibraryFolder("config", 0775)

	// jumppad.config.js
	frontendConfigPath := filepath.Join(configPath, "jumppad.config.js")
	err = p.writeConfig(frontendConfigPath)
	if err != nil {
		return err
	}

	// navigation.jsx
	navigationPath := filepath.Join(configPath, "navigation.jsx")
	err = p.writeNavigation(navigationPath)
	if err != nil {
		return err
	}

	// progress.jsx
	progressPath := filepath.Join(configPath, "progress.jsx")
	err = p.writeProgress(progressPath)
	if err != nil {
		return err
	}

	// /content
	contentPath := utils.LibraryFolder("content", 0775)

	for _, book := range p.config.Content {
		bookPath := filepath.Join(contentPath, book.Name)

		for _, chapter := range book.Chapters {
			chapterPath := filepath.Join(bookPath, chapter.Name)
			os.MkdirAll(chapterPath, 0755)
			os.Chmod(chapterPath, 0755)

			for _, page := range chapter.Pages {
				pageFile := fmt.Sprintf("%s.mdx", page.Name)
				pagePath := filepath.Join(chapterPath, pageFile)
				err := os.WriteFile(pagePath, []byte(page.Content), 0755)
				if err != nil {
					return fmt.Errorf("unable to write page %s to disk at %s", page.Name, pagePath)
				}
			}
		}
	}

	// store a checksum of the content
	cs, err := p.generateContentChecksum()
	if err != nil {
		return fmt.Errorf("unable to generate checksum for content: %s", err)
	}

	p.config.ContentChecksum = cs

	return nil
}

func (p *DocsProvider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.ID)

	// since the content has not been processed we can not reliably determine
	// if the content has changed, so we will assume it has
	return true, nil
}

// check if the content has changed
func (p *DocsProvider) checkChanged() (bool, error) {
	if p.config.ContentChecksum == "" {
		return true, nil
	}

	cs, err := p.generateContentChecksum()
	if err != nil {
		return true, fmt.Errorf("unable to generate checksum for content: %s", err)
	}

	return cs != p.config.ContentChecksum, nil
}

func (p *DocsProvider) generateContentChecksum() (string, error) {
	cs, err := utils.ChecksumFromInterface(p.config.Content)
	if err != nil {
		return "", fmt.Errorf("unable to generate checksum for content: %s", err)
	}

	return cs, nil
}

func (p *DocsProvider) getDefaultPage() string {
	if len(p.config.Content) > 0 {
		book := p.config.Content[0]
		if len(book.Index.Chapters) > 0 {
			chapter := book.Index.Chapters[0]
			if len(chapter.Pages) > 0 {
				page := chapter.Pages[0]
				return page.URI
			}
		}
	}
	return "/"
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

	// ~/.jumppad/library/content
	contentPath := utils.LibraryFolder("content", 0775)

	// mount the content
	cc.Volumes = append(
		cc.Volumes,
		types.Volume{
			Source:      contentPath,
			Destination: "/jumppad/src/pages/docs",
		},
	)

	// ~/.jumppad/library/config
	configPath := utils.LibraryFolder("config", 0775)

	cc.Volumes = append(
		cc.Volumes,
		types.Volume{
			Source:      configPath,
			Destination: "/jumppad/src/config",
		},
	)

	// write the frontend config
	frontendConfigPath := filepath.Join(utils.LibraryFolder("", 0775), "jumppad.config.js")
	err = p.writeConfig(frontendConfigPath)
	if err != nil {
		return err
	}

	cc.Volumes = append(
		cc.Volumes,
		types.Volume{
			Source:      frontendConfigPath,
			Destination: "/jumppad/jumppad.config.mjs",
		},
	)

	// mount the assets
	if p.config.Assets != "" {
		assetsDestination := "/jumppad/public/assets"
		cc.Volumes = append(
			cc.Volumes,
			types.Volume{
				Source:      p.config.Assets,
				Destination: assetsDestination,
			},
		)
	}

	_, err = p.client.CreateContainer(cc)
	return err
}

func (p *DocsProvider) writeProgress(path string) error {
	progress := []Progress{}

	for _, book := range p.config.Content {
		// add progress
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

	content := fmt.Sprintf(`export const progress = %s`, progressJSON)
	err = os.WriteFile(path, []byte(content), 0755)
	if err != nil {
		return fmt.Errorf("unable to write progress to disk at %s", path)
	}

	return nil
}

func (p *DocsProvider) writeNavigation(path string) error {
	indices := []BookIndex{}
	for _, book := range p.config.Content {
		indices = append(indices, book.Index)
	}

	indexJSON, err := json.MarshalIndent(indices, "", " ")
	if err != nil {
		return err
	}

	content := fmt.Sprintf(`export const navigation = %s`, indexJSON)
	err = os.WriteFile(path, []byte(content), 0755)
	if err != nil {
		return fmt.Errorf("unable to write navigation to disk at %s", path)
	}

	return nil
}

func (p *DocsProvider) writeConfig(configPath string) error {
	config := DocsConfig{
		Logo:        p.config.Logo,
		DefaultPath: p.getDefaultPage(),
	}

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
