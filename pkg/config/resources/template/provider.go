package template

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/infinytum/raymond/v2"
	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	sdk "github.com/jumppad-labs/plugin-sdk"
	"github.com/zclconf/go-cty/cty"
)

// Template provider allows parsing and output of file based templates
type TemplateProvider struct {
	config *Template
	log    sdk.Logger
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

func (p *TemplateProvider) Init(cfg htypes.Resource, l sdk.Logger) error {
	c, ok := cfg.(*Template)
	if !ok {
		return fmt.Errorf("unable to initialize Template provider, resource is not of type Template")
	}

	p.config = c
	p.log = l

	return nil
}

// Create a new template
func (p *TemplateProvider) Create() error {
	// check the template is valid
	if p.config.Source == "" {
		return fmt.Errorf("template source empty")
	}

	output := p.config.Source
	if p.config.Variables != nil {

		vars := parseVars(p.config.Variables)

		tmpl, err := raymond.Parse(p.config.Source)
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

		output = result
	}

	// gemerate a checksum from the result
	cs, err := utils.ChecksumFromInterface(output)
	if err != nil {
		return fmt.Errorf("unable to generate checksum for template: %s", err)
	}

	outputExists := false
	if fi, _ := os.Stat(p.config.Destination); fi != nil {
		outputExists = true
	}

	// regenerate the template if it has changed or the file does not exist
	if p.config.Checksum != cs || !outputExists {
		p.log.Info("Generating template", "ref", p.config.Meta.ID, "checksum", p.config.Checksum, "source", p.config.Source, "output", p.config.Destination)

		// set the checksum
		p.config.Checksum = cs

		// if an existing file exists delete it
		if outputExists {
			err = os.RemoveAll(p.config.Destination)
			if err != nil {
				return fmt.Errorf("unable to delete destination file: %s", err)
			}
		}

		err = os.MkdirAll(filepath.Dir(p.config.Destination), os.ModePerm)
		if err != nil {
			return fmt.Errorf("unable to create destination directory for template: %s", err)
		}

		f, err := os.Create(p.config.Destination)
		if err != nil {
			return fmt.Errorf("unable to create destination file for template: %s", err)
		}
		defer f.Close()

		f.WriteString(output)
	}

	return nil
}

func (p *TemplateProvider) Destroy() error {
	if _, err := os.Stat(p.config.Destination); !os.IsNotExist(err) {
		err := os.RemoveAll(p.config.Destination)
		if err != nil {
			p.log.Warn("Unable to delete template file",
				"ref", p.config.Meta.Name,
				"destination", p.config.Destination,
				"error", err)
		}
	}

	return nil
}

// Lookup satisfies the interface method but is not implemented by Template
func (p *TemplateProvider) Lookup() ([]string, error) {
	return []string{}, nil
}

// Refresh causes the template to be destroyed and recreated
func (p *TemplateProvider) Refresh() error {
	p.log.Debug("Refresh Template", "ref", p.config.Meta.ID)

	return p.Create()
}

func (p *TemplateProvider) Changed() (bool, error) {
	return false, nil
}
