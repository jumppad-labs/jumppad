package docs

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
)

const TypeBook string = "book"

type Book struct {
	types.ResourceMetadata `hcl:",remain"`

	Title    string    `hcl:"title" json:"title"`
	Chapters []Chapter `hcl:"chapters" json:"chapters"`

	// Output parameters
	Index BookIndex `hcl:"index,optional" json:"index"`
}

type BookIndex struct {
	Title    string         `hcl:"title,optional" json:"title"`
	Chapters []ChapterIndex `hcl:"chapters,optional" json:"chapters"`
}

func (b *Book) Process() error {
	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(b.ID)
		if r != nil {
			state := r.(*Book)
			b.Index = state.Index
		}
	}

	return nil
}
