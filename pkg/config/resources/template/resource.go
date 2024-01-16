package template

import (
	"os"
	"strings"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/zclconf/go-cty/cty"
)

// TypeTemplate is the resource string for a Template resource
const TypeTemplate string = "template"

// Template allows the process of user defined templates
type Template struct {
	types.ResourceMetadata `hcl:",remain"`

	Source      string               `hcl:"source" json:"source"`                          // Source template to be processed as string
	Destination string               `hcl:"destination" json:"destination"`                // Destination filename to write
	Variables   map[string]cty.Value `hcl:"variables,optional" json:"variables,omitempty"` // Variables to be processed in the template
}

func (t *Template) Process() error {
	t.Destination = utils.EnsureAbsolute(t.Destination, t.ResourceFile)

	// Source can be a file or a template as a string
	// check to see if a valid file before making absolute
	src := t.Source
	absSrc := utils.EnsureAbsolute(src, t.ResourceFile)

	if _, err := os.Stat(absSrc); err == nil {
		// file exists
		t.Source = absSrc
	} else {
		// source is a string, replace line endings
		t.Source = strings.Replace(t.Source, "\r\n", "\n", -1)
	}

	return nil
}
