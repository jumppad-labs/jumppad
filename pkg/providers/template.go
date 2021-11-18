package providers

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/zclconf/go-cty/cty"
)

// Template provider allows parsing and output of file based templates
type Template struct {
	config *config.Template
	log    hclog.Logger
}

// NewTemplate creates a new Local Exec provider
func NewTemplate(c *config.Template, l hclog.Logger) *Template {
	return &Template{c, l}
}

// parseVarse converts a map[string]cty.Value into map[string]interface
// where the interface are generic go types like string, number, bool, slice, map
//
// TODO move this into the parser class and add more robust testing
func parseVars(value map[string]cty.Value) map[string]interface{} {
	vars := map[string]interface{}{}

	for k, v := range value {
		vars[k] = castVar(v)
	}

	return vars
}

func castVar(v cty.Value) interface{} {
	if v.Type() == cty.String {
		return v.AsString()
	} else if v.Type() == cty.Bool {
		return v.True()
	} else if v.Type() == cty.Number {
		return v.AsBigFloat()
	} else if v.Type().IsObjectType() || v.Type().IsMapType() {
		return parseVars(v.AsValueMap())
	} else if v.Type().IsTupleType() || v.Type().IsListType() {
		i := v.ElementIterator()
		vars := []interface{}{}
		for {
			if !i.Next() {
				// cant iterate
				break
			}

			_, value := i.Element()
			vars = append(vars, castVar(value))
		}

		return vars
	}

	return nil
}

// Create a new template
func (c *Template) Create() error {
	c.log.Info("Generating template", "ref", c.config.Name, "output", c.config.Destination)
	c.log.Debug("Template content", "ref", c.config.Name, "source", c.config.Source)

	// check the template is valid
	if c.config.Source == "" {
		return fmt.Errorf("Template source empty")
	}

	if _, ok := c.config.Vars.(*hcl.Attribute); !ok {
		// no variables just write the file
		f, err := os.Create(c.config.Destination)
		if err != nil {
			return fmt.Errorf("Unable to create destination file for template: %s", err)
		}
		defer f.Close()

		c.log.Debug("Template output", "ref", c.config.Name, "destination", c.config.Source)
		_, err = f.WriteString(c.config.Source)

		return err
	}

	// convert the HCL types into Go map[string]interface that can be used by go template
	val, _ := c.config.Vars.(*hcl.Attribute).Expr.Value(config.GetEvalContext())
	m := val.AsValueMap()
	vars := parseVars(m)

	tmpl := template.New("template").Delims("#{{", "}}")

	t, err := tmpl.Parse(c.config.Source)
	if err != nil {
		return fmt.Errorf("Unable to parse template: %s", err)
	}

	bs := bytes.NewBufferString("")
	err = t.Execute(bs, struct{ Vars map[string]interface{} }{Vars: vars})
	if err != nil {
		return fmt.Errorf("Error processing template: %s", err)
	}

	if fi, _ := os.Stat(c.config.Destination); fi != nil {
		err = os.RemoveAll(c.config.Destination)
		if err != nil {
			return fmt.Errorf("Unable to delete destination file: %s", err)
		}
	}

	err = os.MkdirAll(filepath.Dir(c.config.Destination), os.ModePerm)
	if err != nil {
		return fmt.Errorf("Unable to create destination directory for template: %s", err)
	}

	f, err := os.Create(c.config.Destination)
	if err != nil {
		return fmt.Errorf("Unable to create destination file for template: %s", err)
	}
	defer f.Close()

	f.WriteString(bs.String())

	c.log.Debug("Template output", "ref", c.config.Name, "destination", bs.String())

	return nil
}

func (c *Template) Destroy() error {
	if _, err := os.Stat(c.config.Destination); !os.IsNotExist(err) {
		err := os.RemoveAll(c.config.Destination)
		if err != nil {
			c.log.Warn("Unable to delete template file",
				"ref", c.config.Name,
				"destination", c.config.Destination,
				"error", err)
		}
	}

	return nil
}

// Lookup statisfies the interface method but is not implemented by Template
func (c *Template) Lookup() ([]string, error) {
	return []string{}, nil
}
