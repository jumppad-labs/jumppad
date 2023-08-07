package resources

import "github.com/jumppad-labs/hclconfig/types"

type IndexBook struct {
	Title    string         `hcl:"title,optional" json:"title"`
	Chapters []IndexChapter `hcl:"chapters,optional" json:"chapters"`
}

type IndexChapter struct {
	Title string      `hcl:"title,optional" json:"title,omitempty"`
	Pages []IndexPage `hcl:"pages,optional" json:"pages"`
}

type IndexPage struct {
	Title string `hcl:"title,optional" json:"title"`
	URI   string `hcl:"uri,optional" json:"uri"`
}

const TypeBook string = "book"

type Book struct {
	types.ResourceMetadata `hcl:",remain"`

	Title    string   `hcl:"title" json:"title"`
	Chapters []string `hcl:"chapters" json:"chapters"`

	// Output parameters
	Index IndexBook `hcl:"index,optional" json:"index"`
}

func (b *Book) Process() error {
	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(b.ID)
		if r != nil {
			kstate := r.(*Book)
			b.Index = kstate.Index
		}
	}

	return nil
}

const TypeChapter string = "chapter"

type Chapter struct {
	types.ResourceMetadata `hcl:",remain"`

	Prerequisites []string `hcl:"prerequisites,optional" json:"prerequisites"`

	Title string `hcl:"title,optional" json:"title,omitempty"`
	Pages []Page `hcl:"page,block" json:"pages"`

	// Output parameters
	Tasks []string `hcl:"tasks,optional" json:"tasks"`
}

func (c *Chapter) Process() error {
	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(c.ID)
		if r != nil {
			kstate := r.(*Chapter)
			c.Tasks = kstate.Tasks
		}
	}

	return nil
}

type Page struct {
	Name    string            `hcl:"id,label" json:"id"`
	Title   string            `hcl:"title" json:"title"`
	Content string            `hcl:"content" json:"content"`
	Tasks   map[string]string `hcl:"tasks,optional" json:"tasks"`
}
