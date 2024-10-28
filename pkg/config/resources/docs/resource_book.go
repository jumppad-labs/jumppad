package docs

import (
	"github.com/jumppad-labs/hclconfig/types"
)

const TypeBook string = "book"

type Book struct {
	types.ResourceBase `hcl:",remain"`

	Title    string    `hcl:"title" json:"title"`
	Chapters []Chapter `hcl:"chapters" json:"chapters"`
}

func (b *Book) Process() error {

	return nil
}
