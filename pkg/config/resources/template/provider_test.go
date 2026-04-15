package template

import (
	"context"
	"io"
	"os"
	"path"
	"testing"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

var variables = map[string]cty.Value{
	"resource": cty.ObjectVal(map[string]cty.Value{
		"container": cty.ObjectVal(map[string]cty.Value{
			"a": cty.ObjectVal(map[string]cty.Value{
				"output": cty.ObjectVal(map[string]cty.Value{
					"test1": cty.StringVal("foo"),
					"test2": cty.StringVal("bar"),
				}),
			}),
			"b": cty.ObjectVal(map[string]cty.Value{
				"output": cty.ObjectVal(map[string]cty.Value{
					"test3": cty.ListVal([]cty.Value{cty.StringVal("moo"), cty.StringVal("cluck")}),
				}),
			}),
		}),
	}),
}

var variablesWithUnknown = map[string]cty.Value{
	"resource": cty.ObjectVal(map[string]cty.Value{
		"container": cty.ObjectVal(map[string]cty.Value{
			"a": cty.ObjectVal(map[string]cty.Value{
				"output": cty.ObjectVal(map[string]cty.Value{
					"test1": cty.UnknownVal(cty.String),
					"test2": cty.StringVal("bar"),
				}),
			}),
			"b": cty.ObjectVal(map[string]cty.Value{
				"output": cty.ObjectVal(map[string]cty.Value{
					"test3": cty.ListVal([]cty.Value{cty.StringVal("moo"), cty.StringVal("cluck")}),
				}),
			}),
		}),
	}),
}

func setupTemplate(t *testing.T, outputFile string) (*Template, *TemplateProvider) {
	testLogger := logger.NewTestLogger(t)

	template := `
		test1: {{resource.container.a.output.test1}}
		test2: {{resource.container.a.output.test2}}
		test3:
			{{#resource.container.b.output.test3}}
			{{.}}
			{{/resource.container.b.output.test3}}
	`

	c := &Template{
		ResourceBase: types.ResourceBase{Meta: types.Meta{File: "./"}},
		Source:       template,
		Destination:  outputFile,
		Variables:    variables,
	}

	err := c.Process()
	require.NoError(t, err)

	return c, &TemplateProvider{c, testLogger}
}

func TestTemplateHandlesVariableConversion(t *testing.T) {
	tmp := t.TempDir()
	outputFile := path.Join(tmp, "output.hcl")

	_, p := setupTemplate(t, outputFile)

	err := p.Create(context.Background())
	require.NoError(t, err)

	f, err := os.Open(outputFile)
	require.NoError(t, err)
	data, _ := io.ReadAll(f)

	require.Contains(t, string(data), "test1: foo")
	require.Contains(t, string(data), "test2: bar")
	require.Contains(t, string(data), `test3:
			moo
			cluck`)
}

func TestTemplateReturnsErrorWhenVariableUnknown(t *testing.T) {
	tmp := t.TempDir()
	outputFile := path.Join(tmp, "output.hcl")

	c, p := setupTemplate(t, outputFile)
	c.Variables = variablesWithUnknown

	err := p.Create(context.Background())
	require.Error(t, err)
	require.ErrorContains(t, err, "resource.container.a.output.test1")
	require.ErrorContains(t, err, "unknown")
}

func TestTemplateWriteSourceWhenNoVars(t *testing.T) {
	tmp := t.TempDir()
	outputFile := path.Join(tmp, "output.hcl")

	tmpl, provider := setupTemplate(t, outputFile)
	tmpl.Variables = nil

	err := provider.Create(context.Background())
	require.NoError(t, err)

	d, err := os.ReadFile(tmpl.Destination)
	require.NoError(t, err)

	require.Contains(t, string(d), `{{resource.container.a.output.test1}}`)
}

func TestTemplateOverwritesExistingFile(t *testing.T) {
	tmp := t.TempDir()
	outputFile := path.Join(tmp, "output.hcl")

	tmpl, provider := setupTemplate(t, outputFile)

	f, err := os.Create(tmpl.Destination)
	require.NoError(t, err)
	f.WriteString("Some text in the file")
	f.Close()

	err = provider.Create(context.Background())
	require.NoError(t, err)

	d, err := os.ReadFile(tmpl.Destination)
	require.NoError(t, err)

	require.Contains(t, string(d), `test1: foo`)
}

func TestTemplateDestroyRemovesDestination(t *testing.T) {
	tmp := t.TempDir()
	outputFile := path.Join(tmp, "output.hcl")

	tmpl, provider := setupTemplate(t, outputFile)

	f, err := os.Create(tmpl.Destination)
	require.NoError(t, err)
	f.WriteString("test")
	f.Close()

	err = provider.Destroy(context.Background(), false)
	require.NoError(t, err)

	require.NoFileExists(t, tmpl.Destination)
}
