package docs

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

type BookProvider struct {
	config *Book
	log    logger.Logger
}

type Page struct {
	Title string `json:"title"`
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

func (b *BookProvider) Create() error {
	b.log.Info(fmt.Sprintf("Creating %s", b.config.Type), "ref", b.config.Name)

	index := Index{
		Title: b.config.Title,
	}

	libraryPath := utils.GetLibraryFolder("", 0775)
	bookPath := filepath.Join(libraryPath, "content", b.config.Name)

	for _, chapter := range b.config.Chapters {
		ic := IndexChapter{
			Title: chapter.Title,
		}

		chapterPath := filepath.Join(bookPath, chapter.Name)

		os.MkdirAll(chapterPath, 0755)
		os.Chmod(chapterPath, 0755)

		for slug, content := range chapter.Pages {
			abs := utils.EnsureAbsolute(content, chapter.File)

			if _, err := os.Stat(abs); err == nil {
				b, err := os.ReadFile(abs)
				if err != nil {
					return fmt.Errorf("unable to read contents of page %s to disk at %s", slug, abs)
				}

				content = string(b)
			} else {
				// content is a string, replace line endings
				content = strings.Replace(content, "\r\n", "\n", -1)
			}

			// replace task ids
			taskRegex, _ := regexp.Compile("<Task id=\"(?P<id>.*)\">")
			taskMatch := taskRegex.FindAllStringSubmatch(content, -1)
			for _, match := range taskMatch {
				taskID := match[1]
				resourceID := fmt.Sprintf("<Task id=\"%s\">", chapter.Tasks[taskID].ID)
				content = taskRegex.ReplaceAllString(content, resourceID)
			}

			// write the file
			pageFile := fmt.Sprintf("%s.mdx", slug)
			pagePath := filepath.Join(chapterPath, pageFile)
			err := os.WriteFile(pagePath, []byte(content), 0755)
			if err != nil {
				return fmt.Errorf("unable to write page %s to disk at %s", slug, pagePath)
			}

			// capture the title of the page
			title := slug
			titleRegex, _ := regexp.Compile("^#(?P<title>.*)$>")
			titleMatch := titleRegex.FindStringSubmatch(content)
			if len(titleMatch) > 0 {
				title = titleMatch[1]
			}

			ip := IndexPage{
				Title: title,
				URI:   fmt.Sprintf("/%s/%s/%s", b.config.Name, chapter.Name, slug),
			}

			ic.Pages = append(ic.Pages, ip)
		}

		index.Chapters = append(index.Chapters, ic)
	}

	b.config.Index = index

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

// func (p *BookProvider) writePage(chapterPath string, page Page) error {
// 	os.MkdirAll(chapterPath, 0755)
// 	os.Chmod(chapterPath, 0755)

// 	if len(page.Tasks) > 0 {
// 		r, _ := regexp.Compile("<Task id=\"(?P<id>.*)\">")
// 		match := r.FindStringSubmatch(page.Content)
// 		result := map[string]string{}
// 		for i, name := range r.SubexpNames() {
// 			if i != 0 && name != "" {
// 				result[name] = match[i]
// 			}
// 		}

// 		if len(match) > 0 {
// 			taskID := result["id"]
// 			resourceID := fmt.Sprintf("<Task id=\"%s\">", page.Tasks[taskID].ID)
// 			page.Content = r.ReplaceAllString(page.Content, resourceID)
// 		}
// 	}

// 	pageFile := fmt.Sprintf("%s.mdx", page.Name)
// 	pagePath := filepath.Join(chapterPath, pageFile)
// 	err := os.WriteFile(pagePath, []byte(page.Content), 0755)
// 	if err != nil {
// 		return fmt.Errorf("unable to write page %s to disk at %s", page.Name, pagePath)
// 	}

// 	return nil
// }
