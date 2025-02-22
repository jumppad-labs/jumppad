package docs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/container"
	"github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/network"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	sdk "github.com/jumppad-labs/plugin-sdk"
	"github.com/mohae/deepcopy"
)

const docsImageName = "ghcr.io/jumppad-labs/docs"
const docsVersion = "v0.5.1"

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

type BookIndex struct {
	Title    string         `hcl:"title,optional" json:"title"`
	Chapters []ChapterIndex `hcl:"chapters,optional" json:"chapters"`
}

type ChapterIndex struct {
	Title string             `hcl:"title,optional" json:"title,omitempty"`
	Pages []ChapterIndexPage `hcl:"pages" json:"pages"`
}

type ChapterIndexPage struct {
	Title string `hcl:"title" json:"title"`
	URI   string `hcl:"uri" json:"uri"`
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
func (p *DocsProvider) Create(ctx context.Context) error {
	if ctx.Err() != nil {
		p.log.Debug("Context is cancelled, skipping create", "ref", p.config.Meta.ID)
		return nil
	}

	p.log.Info("Creating Documentation", "ref", p.config.Meta.ID)
	err := p.generateDocs()
	if err != nil {
		return err
	}

	// write the content
	return p.createDocsContainer()
}

// Destroy the documentation container
func (p *DocsProvider) Destroy(ctx context.Context, force bool) error {
	if ctx.Err() != nil {
		p.log.Debug("Context is cancelled, skipping destroy", "ref", p.config.Meta.ID)
		return nil
	}

	p.log.Info("Destroy Documentation", "ref", p.config.Meta.ID)

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

func (p *DocsProvider) Refresh(ctx context.Context) error {
	if ctx.Err() != nil {
		p.log.Debug("Context is cancelled, skipping refresh", "ref", p.config.Meta.ID)
		return nil
	}

	p.log.Debug("Refresh Docs", "ref", p.config.Meta.ID, "checksum", p.config.ContentChecksum)

	changed, err := p.checkChanged()
	if err != nil {
		return fmt.Errorf("unable to check if content has changed: %s", err)
	}

	// no changes return
	if !changed {
		return nil
	}

	return p.generateDocs()
}

func (p *DocsProvider) Changed() (bool, error) {
	return p.checkChanged()
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

	if cs != p.config.ContentChecksum {
		p.log.Debug("Content changed", "ref", p.config.Meta.ID, "checksum", cs, "old", p.config.ContentChecksum)
		return true, nil
	}

	return false, nil
}

func (p *DocsProvider) generateContentChecksum() (string, error) {
	books := deepcopy.Copy(p.config.Content).([]Book)

	// replace the processed checksum as this will cause the content
	// to always be different
	for i := range books {
		books[i].Meta.Checksum.Processed = ""
		for c := range books[i].Chapters {
			books[i].Chapters[c].Meta.Checksum.Processed = ""
		}
	}

	cs, err := utils.ChecksumFromInterface(books)
	if err != nil {
		return "", fmt.Errorf("unable to generate checksum for content: %s", err)
	}

	return cs, nil
}

func (p *DocsProvider) createDocsContainer() error {
	// set the FQDN
	fqdn := utils.FQDN(p.config.Meta.Name, p.config.Meta.Module, p.config.Meta.Type)
	p.config.ContainerName = fqdn

	// create the container config
	cc := &types.Container{
		Name: fqdn,
	}

	cc.Networks = p.config.Networks.ToClientNetworkAttachments()

	if len(cc.Networks) == 0 {
		cc.Networks = append(cc.Networks, types.NetworkAttachment{
			ID:   network.DefaultNetworkID,
			Name: network.DefaultNetworkName,
		})
	}

	cc.Image = &types.Image{Name: fmt.Sprintf("%s:%s", docsImageName, docsVersion)}
	cc.MaxRestartCount = -1

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

	// write the temp file for the config
	frontendConfigPath := filepath.Join(configPath, "jumppad.config.js")

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

func (p *DocsProvider) generateDocs() error {
	p.log.Info("Refresh Docs", "ref", p.config.Meta.ID)

	// refresh content on disk
	configPath := utils.LibraryFolder("config", 0775)

	// jumppad.config.js

	// navigation.jsx
	navigationPath := filepath.Join(configPath, "navigation.jsx")
	indexPage, err := p.writeNavigation(navigationPath)
	if err != nil {
		return err
	}

	// progress.jsx
	progressPath := filepath.Join(configPath, "progress.jsx")
	err = p.writeProgress(progressPath)
	if err != nil {
		return err
	}

	frontendConfigPath := filepath.Join(configPath, "jumppad.config.js")
	err = p.writeConfig(frontendConfigPath, indexPage)
	if err != nil {
		return err
	}

	// /content
	contentPath := utils.LibraryFolder("content", 0775)

	for _, book := range p.config.Content {
		bookPath := filepath.Join(contentPath, book.Meta.Name)

		for _, chapter := range book.Chapters {
			chapterPath := filepath.Join(bookPath, chapter.Meta.Name)
			os.MkdirAll(chapterPath, 0755)
			os.Chmod(chapterPath, 0755)

			for _, page := range chapter.Pages {
				err := p.processPage(chapterPath, chapter, page)
				if err != nil {
					return err
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

	p.log.Debug("Content written", "ref", p.config.Meta.ID, "checksum", p.config.ContentChecksum)

	return nil
}

func (p *DocsProvider) processPage(chapterPath string, chapter Chapter, page Page) error {
	content := strings.Replace(page.Content, "\r\n", "\n", -1)

	// replace task ids
	taskRegex, _ := regexp.Compile("<Task id=\"(?P<id>.*)\">")
	taskMatch := taskRegex.FindAllStringSubmatch(page.Content, -1)
	for _, match := range taskMatch {
		taskID := match[1]
		resourceID := fmt.Sprintf("<Task id=\"%s\">", chapter.Tasks[taskID].Meta.ID)
		content = taskRegex.ReplaceAllString(page.Content, resourceID)
	}

	pageFile := fmt.Sprintf("%s.mdx", page.Name)
	pagePath := filepath.Join(chapterPath, pageFile)
	err := os.WriteFile(pagePath, []byte(content), 0755)
	if err != nil {
		return fmt.Errorf("unable to write page %s to disk at %s", page.Name, pagePath)
	}

	return nil
}

func (p *DocsProvider) writeProgress(path string) error {
	progress := []Progress{}

	for _, book := range p.config.Content {
		// add progress
		for _, chapter := range book.Chapters {
			for _, task := range chapter.Tasks {
				p := Progress{
					ID:            task.Meta.ID,
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

// writes the navigation config and returns the index page for the config
func (p *DocsProvider) writeNavigation(path string) (string, error) {
	indexPage := "/"

	indices := []BookIndex{}
	for b, book := range p.config.Content {
		bookIndex := BookIndex{
			Title:    book.Title,
			Chapters: []ChapterIndex{},
		}

		for c, chapter := range book.Chapters {
			chapterIndex := ChapterIndex{
				Title: chapter.Title,
				Pages: []ChapterIndexPage{},
			}

			for p, page := range chapter.Pages {
				pageIndex := ChapterIndexPage{
					Title: page.Name,
					URI:   fmt.Sprintf("/docs/%s/%s/%s", book.Meta.Name, chapter.Meta.Name, page.Name),
				}

				// if this is the first book and chapter and page
				// set the index page
				if b == 0 && c == 0 && p == 0 {
					indexPage = pageIndex.URI
				}

				// get the title from the heading of the page
				titleRegex, _ := regexp.Compile(`^#\s?(?P<title>.*)`)
				titleMatch := titleRegex.FindStringSubmatch(page.Content)
				if len(titleMatch) > 0 {
					pageIndex.Title = titleMatch[1]
				}

				chapterIndex.Pages = append(chapterIndex.Pages, pageIndex)
			}

			bookIndex.Chapters = append(bookIndex.Chapters, chapterIndex)
		}

		indices = append(indices, bookIndex)
	}

	indexJSON, err := json.MarshalIndent(indices, "", " ")
	if err != nil {
		return "", err
	}

	content := fmt.Sprintf(`export const navigation = %s`, indexJSON)
	err = os.WriteFile(path, []byte(content), 0755)
	if err != nil {
		return "", fmt.Errorf("unable to write navigation to disk at %s", path)
	}

	return indexPage, nil
}

func (p *DocsProvider) writeConfig(configPath, indexPage string) error {
	config := DocsConfig{
		Logo:        p.config.Logo,
		DefaultPath: indexPage,
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
