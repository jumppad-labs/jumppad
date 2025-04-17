package template

import (
	"os"
	"strings"

	"github.com/instruqt/jumppad/pkg/config"
	"github.com/instruqt/jumppad/pkg/utils"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/zclconf/go-cty/cty"
)

// TypeTemplate is the resource string for a Template resource
const TypeTemplate string = "template"

// Template allows the process of user defined templates
type Template struct {
	types.ResourceBase `hcl:",remain"`

	Source      string               `hcl:"source" json:"source"`                          // Source template to be processed as string
	Destination string               `hcl:"destination" json:"destination"`                // Destination filename to write
	Variables   map[string]cty.Value `hcl:"variables,optional" json:"variables,omitempty"` // Variables to be processed in the template

	Checksum string `hcl:"checksum,optional" json:"checksum,omitempty"` // Checksum of the parsed template
}

func (t *Template) Process() error {
	t.Destination = utils.EnsureAbsolute(t.Destination, t.Meta.File)

	// Source can be a file or a template as a string
	// check to see if a valid file before making absolute
	src := t.Source
	absSrc := utils.EnsureAbsolute(src, t.Meta.File)

	if _, err := os.Stat(absSrc); err == nil {
		// file exists
		t.Source = absSrc
	} else {
		// source is a string, replace line endings
		t.Source = strings.Replace(t.Source, "\r\n", "\n", -1)
	}

	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(t.Meta.ID)
		if r != nil {
			kstate := r.(*Template)
			t.Checksum = kstate.Checksum
		}
	}

	return nil
}
