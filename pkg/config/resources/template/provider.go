package template

import (
	"context"
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

var _ sdk.Provider = &TemplateProvider{}

// Template provider allows parsing and output of file based templates
type TemplateProvider struct {
	config *Template
	log    sdk.Logger
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
func (p *TemplateProvider) Create(ctx context.Context) error {
	if ctx.Err() != nil {
		p.log.Debug("Context cancelled, skipping create", "ref", p.config.Meta.ID)
		return nil
	}

	// check the template is valid
	if p.config.Source == "" {
		return fmt.Errorf("template source empty")
	}

	output := p.config.Source
	if p.config.Variables != nil {

		vars, err := parseVars(p.config.Variables, "")
		if err != nil {
			return fmt.Errorf("error parsing template variables: %s", err)
		}

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

func (p *TemplateProvider) Destroy(ctx context.Context, force bool) error {
	if ctx.Err() != nil {
		p.log.Debug("Context cancelled, skipping destroy", "ref", p.config.Meta.ID)
		return nil
	}

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
func (p *TemplateProvider) Refresh(ctx context.Context) error {
	if ctx.Err() != nil {
		p.log.Debug("Context cancelled, skipping refresh", "ref", p.config.Meta.ID)
		return nil
	}

	p.log.Debug("Refresh Template", "ref", p.config.Meta.ID)

	return p.Create(ctx)
}

func (p *TemplateProvider) Changed() (bool, error) {
	return false, nil
}

// parseVars converts a map[string]cty.Value into map[string]any
// where the values are generic go types like string, number, bool, slice, map.
// path is the dotted key path built up during recursion so errors can point
// at the exact offending variable.
//
// TODO move this into the parser class and add more robust testing
func parseVars(value map[string]cty.Value, path string) (map[string]any, error) {
	vars := map[string]any{}

	for k, v := range value {
		child := k
		if path != "" {
			child = path + "." + k
		}

		cast, err := castVar(v, child)
		if err != nil {
			return nil, err
		}

		vars[k] = cast
	}

	return vars, nil
}

func castVar(v cty.Value, path string) (any, error) {
	if v.IsNull() {
		return nil, nil
	}

	// hclconfig seeds typed-but-unknown placeholder values when a template
	// references a variable that does not resolve. Surface this as an error
	// rather than panicking inside AsString/True/AsBigFloat below.
	if !v.IsKnown() {
		return nil, fmt.Errorf(
			"template variable %q is unknown, check that your template references are correct",
			path,
		)
	}

	if v.Type() == cty.String {
		return v.AsString(), nil
	} else if v.Type() == cty.Bool {
		return v.True(), nil
	} else if v.Type() == cty.Number {
		return v.AsBigFloat(), nil
	} else if v.Type().IsObjectType() || v.Type().IsMapType() {
		return parseVars(v.AsValueMap(), path)
	} else if v.Type().IsTupleType() || v.Type().IsListType() {
		i := v.ElementIterator()
		vars := []any{}
		idx := 0
		for {
			if !i.Next() {
				break
			}

			_, value := i.Element()
			cast, err := castVar(value, fmt.Sprintf("%s[%d]", path, idx))
			if err != nil {
				return nil, err
			}

			vars = append(vars, cast)
			idx++
		}

		return vars, nil
	}

	return nil, nil
}
