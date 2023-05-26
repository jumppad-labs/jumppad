package resources

import (
	"os"

	"github.com/jumppad-labs/hclconfig/types"
)

// TypeTemplate is the resource string for a Template resource
const TypeTemplate string = "template"

// Template allows the process of user defined templates
type Template struct {
	types.ResourceMetadata `hcl:",remain"`

	Source      string      `hcl:"source" json:"source"`                          // Source template to be processed as string
	Destination string      `hcl:"destination" json:"destination"`                // Destination filename to write
	Variables   interface{} `hcl:"variables,optional" json:"variables,omitempty"` // Variables to be processed in the template
}

func (t *Template) Process() error {
	t.Destination = ensureAbsolute(t.Destination, t.File)

	// Source can be a file or a template as a string
	// check to see if a valid file before making absolute
	src := t.Source
	absSrc := ensureAbsolute(src, t.File)

	if _, err := os.Stat(absSrc); err == nil {
		// file exists
		t.Source = absSrc
	}

	return nil
}
