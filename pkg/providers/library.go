package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

type Book struct {
	config *resources.Book
	log    hclog.Logger
}

func NewBook(b *resources.Book, l hclog.Logger) *Book {
	return &Book{b, l}
}

func (b *Book) Create() error {
	b.log.Info(fmt.Sprintf("Creating %s", strings.Title(string(b.config.Metadata().Type))), "ref", b.config.Metadata().Name)

	book := resources.IndexBook{
		Title: b.config.Title,
	}

	libraryPath := utils.GetLibraryFolder("", 0775)
	bookPath := filepath.Join(libraryPath, "content", b.config.Name)

	for _, bc := range b.config.Chapters {
		cr, err := b.config.ParentConfig.FindResource(bc)
		if err != nil {
			return fmt.Errorf("Unable to create book %s, could not find chapter %s", b.config.Metadata().Name, bc)
		}

		c := cr.(*resources.Chapter)

		chapterPath := filepath.Join(bookPath, c.Name)
		os.MkdirAll(chapterPath, 0755)
		os.Chmod(chapterPath, 0755)

		chapter := resources.IndexChapter{
			Title: c.Title,
		}

		for _, p := range c.Pages {
			if len(p.Tasks) > 0 {
				r, _ := regexp.Compile("<Task id=\"(?P<id>.*)\">")
				match := r.FindStringSubmatch(p.Content)
				result := map[string]string{}
				for i, name := range r.SubexpNames() {
					if i != 0 && name != "" {
						result[name] = match[i]
					}
				}

				if len(match) > 0 {
					taskID := result["id"]
					resourceID := fmt.Sprintf("<Task id=\"%s\">", p.Tasks[taskID])
					p.Content = r.ReplaceAllString(p.Content, resourceID)
				}
			}

			pageFile := fmt.Sprintf("%s.mdx", p.Name)
			pagePath := filepath.Join(chapterPath, pageFile)
			err := os.WriteFile(pagePath, []byte(p.Content), 0755)
			if err != nil {
				return fmt.Errorf("Unable to write page %s to disk at %s", p.Name, pagePath)
			}

			page := resources.IndexPage{
				Title: p.Title,
				URI:   filepath.Join("/", b.config.Name, c.Name, p.Name),
			}

			chapter.Pages = append(chapter.Pages, page)
		}

		book.Chapters = append(book.Chapters, chapter)
	}

	b.config.Index = book

	return nil
}

func (b *Book) Destroy() error {
	return nil
}

func (b *Book) Lookup() ([]string, error) {
	return nil, nil
}

func (b *Book) Refresh() error {
	return nil
}

type Chapter struct {
	config types.Resource
	log    hclog.Logger
}

func NewChapter(c types.Resource, l hclog.Logger) *Chapter {
	return &Chapter{c, l}
}

func (c *Chapter) Create() error {
	c.log.Info(fmt.Sprintf("Creating %s", strings.Title(string(c.config.Metadata().Type))), "ref", c.config.Metadata().Name)

	return nil
}

func (c *Chapter) Destroy() error {
	return nil
}

func (c *Chapter) Lookup() ([]string, error) {
	return nil, nil
}

func (c *Chapter) Refresh() error {
	return nil
}
