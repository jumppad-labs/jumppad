package providers

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/shipyard-run/shipyard/pkg/config/resources"
	"github.com/zclconf/go-cty/cty"
)

// Template provider allows parsing and output of file based templates
type Template struct {
	config *resources.Template
	log    hclog.Logger
}

// NewTemplate creates a new Local Exec provider
func NewTemplate(c *resources.Template, l hclog.Logger) *Template {
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

	if _, ok := c.config.Vars.(*hcl.Attribute); !ok {
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

	val, _ := c.config.Vars.(*hcl.Attribute).Expr.Value(&hcl.EvalContext{})
	m := val.AsValueMap()
	vars := parseVars(m)

	c.config.InternalVars = vars

	tmpl := template.New("template").Delims("#{{", "}}")
	tmpl.Funcs(template.FuncMap{
		"file":  templateFuncFile,
		"quote": templateFuncQuote,
		"trim":  templateFuncTrim,
	})

	t, err := tmpl.Parse(c.config.Source)
	if err != nil {
		return fmt.Errorf("unable to parse template: %s", err)
	}

	bs := bytes.NewBufferString("")
	err = t.Execute(bs, struct{ Vars map[string]interface{} }{Vars: c.config.InternalVars})
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

// Lookup satisfies the interface method but is not implemented by Template
func (c *Template) Lookup() ([]string, error) {
	return []string{}, nil
}

// wraps the given string in quotes and returns
func templateFuncQuote(in string) string {
	return fmt.Sprintf(`"%s"`, in)
}

// trims whitespace from the given string
func templateFuncTrim(in string) string {
	return strings.TrimSpace(in)
}

// template function that reads a file an returns the string contents
func templateFuncFile(path string) string {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err.Error()
	}

	return string(data)
}
