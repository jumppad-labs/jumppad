package resources

import "github.com/jumppad-labs/hclconfig/types"

type IndexBook struct {
	Title    string         `json:"title"`
	Chapters []IndexChapter `json:"chapters"`
}

type IndexChapter struct {
	Title string      `json:"title,omitempty"`
	Pages []IndexPage `json:"pages"`
}

type IndexPage struct {
	Title string `json:"title"`
	URI   string `json:"uri"`
}

const TypeBook string = "book"

type Book struct {
	types.ResourceMetadata `hcl:",remain"`

	Title    string   `hcl:"title" json:"title"`
	Chapters []string `hcl:"chapters" json:"chapters"`

	// Output parameters
	Index IndexBook `hcl:"index,optional" json:"index"`
}

const TypeChapter string = "chapter"

type Chapter struct {
	types.ResourceMetadata `hcl:",remain"`

	Prerequisites []string `hcl:"prerequisites,optional" json:"prerequisites,omitempty"`

	Title string `hcl:"title,optional" json:"title,omitempty"`
	Pages []Page `hcl:"page,block" json:"pages"`
}

type Page struct {
	Name    string            `hcl:"id,label" json:"id"`
	Title   string            `hcl:"title" json:"title"`
	Content string            `hcl:"content" json:"content"`
	Tasks   map[string]string `hcl:"tasks,optional" json:"tasks"`
}
