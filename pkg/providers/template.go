package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/mailgun/raymond/v2"
	"github.com/zclconf/go-cty/cty"
)

// Template provider allows parsing and output of file based templates
type Template struct {
	config *resources.Template
	log    clients.Logger
}

// NewTemplate creates a new Local Exec provider
func NewTemplate(c *resources.Template, l clients.Logger) *Template {
	return &Template{c, l}
}

// parseVars converts a map[string]cty.Value into map[string]interface
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
	if v.IsNull() {
		return nil
	}

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
	c.log.Info("Generating template", "ref", c.config.ID, "output", c.config.Destination)
	c.log.Debug("Template content", "ref", c.config.ID, "source", c.config.Source)

	// check the template is valid
	if c.config.Source == "" {
		return fmt.Errorf("template source empty")
	}

	if c.config.Variables == nil {
		// no variables just write the file
		f, err := os.Create(c.config.Destination)
		if err != nil {
			return fmt.Errorf("unable to create destination file for template: %s", err)
		}
		defer f.Close()

		c.log.Debug("Template output", "ref", c.config.Name, "destination", c.config.Source)
		_, err = f.WriteString(c.config.Source)

		return err
	}

	vars := parseVars(c.config.Variables)

	tmpl, err := raymond.Parse(c.config.Source)
	if err != nil {
		return fmt.Errorf("error parsing template: %s", err)
	}

	tmpl.RegisterHelpers(map[string]interface{}{
		"quote": func(in string) string {
			return fmt.Sprintf(`"%s"`, in)
		},
		"trim": func(in string) string {
			return strings.TrimSpace(in)
		},
	})

	result, err := tmpl.Exec(vars)
	if err != nil {
		return fmt.Errorf("error processing template: %s", err)
	}

	if fi, _ := os.Stat(c.config.Destination); fi != nil {
		err = os.RemoveAll(c.config.Destination)
		if err != nil {
			return fmt.Errorf("unable to delete destination file: %s", err)
		}
	}

	err = os.MkdirAll(filepath.Dir(c.config.Destination), os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to create destination directory for template: %s", err)
	}

	f, err := os.Create(c.config.Destination)
	if err != nil {
		return fmt.Errorf("unable to create destination file for template: %s", err)
	}
	defer f.Close()

	f.WriteString(result)

	c.log.Debug("Template output", "ref", c.config.Name, "destination", c.config.Destination, "result", result)

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

// Lookup satisfies the interface method but is not implemented by Template
func (c *Template) Lookup() ([]string, error) {
	return []string{}, nil
}

// Refresh causes the template to be destroyed and recreated
func (c *Template) Refresh() error {
	c.log.Debug("Refresh Template", "ref", c.config.ID)

	c.Destroy()
	return c.Create()
}

func (c *Template) Changed() (bool, error) {

	return false, nil
}
