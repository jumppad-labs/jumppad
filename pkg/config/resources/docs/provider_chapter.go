package docs

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	htypes "github.com/jumppad-labs/hclconfig/types"
	sdk "github.com/jumppad-labs/plugin-sdk"
)

type ChapterProvider struct {
	config *Chapter
	log    sdk.Logger
}

func (p *ChapterProvider) Init(cfg htypes.Resource, l sdk.Logger) error {
	c, ok := cfg.(*Chapter)
	if !ok {
		return fmt.Errorf("unable to initialize Chapter provider, resource is not of type Chapter")
	}

	p.config = c
	p.log = l

	return nil
}

func (p *ChapterProvider) Create(ctx context.Context) error {
	if ctx.Err() != nil {
		p.log.Debug("Context is cancelled, skipping create", "ref", p.config.Meta.ID)
		return nil
	}

	index := ChapterIndex{
		Title: p.config.Title,
	}

	for i, page := range p.config.Pages {
		page.Content = strings.Replace(page.Content, "\r\n", "\n", -1)

		// replace task ids
		taskRegex, _ := regexp.Compile("<Task id=\"(?P<id>.*)\">")
		taskMatch := taskRegex.FindAllStringSubmatch(page.Content, -1)
		for _, match := range taskMatch {
			taskID := match[1]
			resourceID := fmt.Sprintf("<Task id=\"%s\">", p.config.Tasks[taskID].Meta.ID)
			page.Content = taskRegex.ReplaceAllString(page.Content, resourceID)
		}

		// replace the file with the content
		p.config.Pages[i].Content = page.Content

		// capture the title of the page
		title := page.Name
		titleRegex, _ := regexp.Compile(`^#\s?(?P<title>.*)`)
		titleMatch := titleRegex.FindStringSubmatch(page.Content)
		if len(titleMatch) > 0 {
			title = titleMatch[1]
		}

		page := ChapterIndexPage{
			Title: title,
			URI:   fmt.Sprintf("%s/%s", p.config.Meta.Name, page.Name),
		}

		index.Pages = append(index.Pages, page)
	}

	p.config.Index = index

	return nil
}

func (p *ChapterProvider) Destroy(ctx context.Context, force bool) error {
	return nil
}

func (p *ChapterProvider) Lookup() ([]string, error) {
	return nil, nil
}

func (p *ChapterProvider) Refresh(ctx context.Context) error {
	p.Create(context.Background()) // always generate content

	return nil
}

func (p *ChapterProvider) Changed() (bool, error) {
	return false, nil
}
