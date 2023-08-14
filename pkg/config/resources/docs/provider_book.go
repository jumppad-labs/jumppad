package docs

import (
	"fmt"

	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
)

type BookProvider struct {
	config *Book
	log    logger.Logger
}

func (p *BookProvider) Init(cfg htypes.Resource, l logger.Logger) error {
	c, ok := cfg.(*Book)
	if !ok {
		return fmt.Errorf("unable to initialize Book provider, resource is not of type Book")
	}

	p.config = c
	p.log = l

	return nil
}

func (p *BookProvider) Create() error {
	p.log.Info(fmt.Sprintf("Creating %s", p.config.Type), "ref", p.config.Name)

	index := BookIndex{
		Title: p.config.Title,
	}

	// prepend the book name to the path of pages
	for _, chapter := range p.config.Chapters {
		for slug, page := range chapter.Index.Pages {
			chapter.Index.Pages[slug].URI = fmt.Sprintf("/docs/%s/%s", p.config.Name, page.URI)
		}

		index.Chapters = append(index.Chapters, chapter.Index)
	}

	p.config.Index = index

	return nil
}

func (p *BookProvider) Destroy() error {
	return nil
}

func (p *BookProvider) Lookup() ([]string, error) {
	return nil, nil
}

func (p *BookProvider) Refresh() error {
	p.log.Debug("Refresh Book", "ref", p.config.ID)

	p.Destroy()
	p.Create()

	return nil
}

func (p *BookProvider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.ID)
	return false, nil
}
