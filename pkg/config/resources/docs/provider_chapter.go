package docs

import (
	"fmt"
	"regexp"
	"strings"

	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
)

type ChapterProvider struct {
	config *Chapter
	log    logger.Logger
}

func (p *ChapterProvider) Init(cfg htypes.Resource, l logger.Logger) error {
	c, ok := cfg.(*Chapter)
	if !ok {
		return fmt.Errorf("unable to initialize Chapter provider, resource is not of type Chapter")
	}

	p.config = c
	p.log = l

	return nil
}

func (p *ChapterProvider) Create() error {
	p.log.Info(fmt.Sprintf("Creating %s", p.config.Type), "ref", p.config.ID)

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
			resourceID := fmt.Sprintf("<Task id=\"%s\">", p.config.Tasks[taskID].ID)
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
			URI:   fmt.Sprintf("%s/%s", p.config.Name, page.Name),
		}

		index.Pages = append(index.Pages, page)
	}

	p.config.Index = index

	return nil
}

func (p *ChapterProvider) Destroy() error {
	return nil
}

func (p *ChapterProvider) Lookup() ([]string, error) {
	return nil, nil
}

func (p *ChapterProvider) Refresh() error {
	p.log.Debug("Refresh Chapter", "ref", p.config.ID)

	p.Destroy()
	p.Create()

	return nil
}

func (p *ChapterProvider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.ID)

	return false, nil
}
