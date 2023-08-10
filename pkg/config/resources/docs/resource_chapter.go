package docs

import (
	"github.com/jumppad-labs/hclconfig/types"
)

const TypeChapter string = "chapter"

type Chapter struct {
	types.ResourceMetadata `hcl:",remain"`

	Prerequisites []string `hcl:"prerequisites,optional" json:"prerequisites"`

	Title string            `hcl:"title,optional" json:"title,omitempty"`
	Pages map[string]string `hcl:"pages,optional" json:"pages"`
	Tasks map[string]Task   `hcl:"tasks,optional" json:"tasks"`
}
