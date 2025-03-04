package docs

import (
	"github.com/jumppad-labs/hclconfig/types"
)

const TypeChapter string = "chapter"

/*
@resource
*/
type Chapter struct {
	/*
	 embedded type holding name, etc

	 @ignore
	*/
	types.ResourceBase `hcl:",remain"`

	Prerequisites []string `hcl:"prerequisites,optional" json:"prerequisites"`

	Title string          `hcl:"title,optional" json:"title,omitempty"`
	Pages []Page          `hcl:"page,block" json:"pages"`
	Tasks map[string]Task `hcl:"tasks,optional" json:"tasks"`
}

type Page struct {
	Name    string `hcl:"name,label" json:"name"`
	Content string `hcl:"content" json:"content"`
}

func (c *Chapter) Process() error {

	return nil
}
