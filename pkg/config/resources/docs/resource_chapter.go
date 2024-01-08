package docs

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
)

const TypeChapter string = "chapter"

type Chapter struct {
	types.ResourceMetadata `hcl:",remain"`

	Prerequisites []string `hcl:"prerequisites,optional" json:"prerequisites"`

	Title string          `hcl:"title,optional" json:"title,omitempty"`
	Pages []Page          `hcl:"page,block" json:"pages"`
	Tasks map[string]Task `hcl:"tasks,optional" json:"tasks"`

	Index ChapterIndex `hcl:"index,optional" json:"index"`
}

type Page struct {
	Name    string `hcl:"name,label" json:"name"`
	Content string `hcl:"content" json:"content"`
}

type ChapterIndex struct {
	Title string             `hcl:"title,optional" json:"title,omitempty"`
	Pages []ChapterIndexPage `hcl:"pages" json:"pages"`
}

type ChapterIndexPage struct {
	Title string `hcl:"title" json:"title"`
	URI   string `hcl:"uri" json:"uri"`
}

func (c *Chapter) Process() error {
	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(c.ResourceID)
		if r != nil {
			state := r.(*Chapter)
			c.Index = state.Index
		}
	}

	return nil
}
