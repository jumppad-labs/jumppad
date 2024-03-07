package docs

import (
	"context"
	"fmt"

	htypes "github.com/jumppad-labs/hclconfig/types"
	sdk "github.com/jumppad-labs/plugin-sdk"
)

type BookProvider struct {
	config *Book
	log    sdk.Logger
}

func (p *BookProvider) Init(cfg htypes.Resource, l sdk.Logger) error {
	c, ok := cfg.(*Book)
	if !ok {
		return fmt.Errorf("unable to initialize Book provider, resource is not of type Book")
	}

	p.config = c
	p.log = l

	return nil
}

func (p *BookProvider) Create(ctx context.Context) error {
	if ctx.Err() != nil {
		p.log.Debug("Context is cancelled, skipping create", "ref", p.config.Meta.ID)
		return nil
	}

	index := BookIndex{
		Title: p.config.Title,
	}

	// prepend the book name to the path of pages
	for _, chapter := range p.config.Chapters {
		for slug, page := range chapter.Index.Pages {
			chapter.Index.Pages[slug].URI = fmt.Sprintf("/docs/%s/%s", p.config.Meta.Name, page.URI)
		}

		index.Chapters = append(index.Chapters, chapter.Index)
	}

	p.config.Index = index

	return nil
}

func (p *BookProvider) Destroy(ctx context.Context, force bool) error {
	return nil
}

func (p *BookProvider) Lookup() ([]string, error) {
	return nil, nil
}

func (p *BookProvider) Refresh(ctx context.Context) error {
	p.Create(ctx)

	return nil
}

func (p *BookProvider) Changed() (bool, error) {
	return false, nil
}
