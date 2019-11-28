package config

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclparse"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

var ctx *hcl.EvalContext

// Config defines the stack config
type Config struct {
	Clusters []*Cluster
	Networks []*Network
}

// ParseFolder for config entries
func ParseFolder(folder string) (*Config, error) {
	ctx = buildContext()

	abs, _ := filepath.Abs(folder)
	c := &Config{}

	// current folder
	files, err := filepath.Glob(path.Join(abs, "*.hcl"))
	if err != nil {
		fmt.Println("err")
		return c, err
	}

	// sub folders
	filesDir, err := filepath.Glob(path.Join(abs, "**/*.hcl"))
	if err != nil {
		fmt.Println("err")
		return c, err
	}

	files = append(files, filesDir...)

	for _, f := range files {
		err := c.ParseHCLFile(f)
		if err != nil {
			return c, err
		}
		/*

			for k, v := range conf.Pipes {
				c.Pipes[k] = v
			}
			for k, v := range conf.Inputs {
				c.Inputs[k] = v
			}
			for k, v := range conf.Outputs {
				c.Outputs[k] = v
			}
			for k, v := range conf.ConnectionPools {
				c.ConnectionPools[k] = v
			}
		*/
	}

	return c, nil
}

// ParseHCLFile parses a config file and adds it to the config
func (c *Config) ParseHCLFile(file string) error {
	fmt.Println("Parsing", file)
	parser := hclparse.NewParser()

	f, diag := parser.ParseHCLFile(file)
	if diag.HasErrors() {
		return errors.New(diag.Error())
	}

	body, ok := f.Body.(*hclsyntax.Body)
	if !ok {
		return errors.New("Error getting body")
	}

	for _, b := range body.Blocks {
		switch b.Type {
		case "cluster":
			cl := &Cluster{}
			cl.name = b.Labels[0]

			err := decodeBody(b, cl)
			if err != nil {
				return err
			}

			c.Clusters = append(c.Clusters, cl)

			/*
				case "input":
					fallthrough
				case "output":
					if err := processBody(config, b); err != nil {
						return config, err
					}

				case "pipe":
					if err := processPipe(config, b); err != nil {
						return config, err
					}
			*/
		}
	}

	return nil
}

func buildContext() *hcl.EvalContext {
	var EnvFunc = function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name:             "env",
				Type:             cty.String,
				AllowDynamicType: true,
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			return cty.StringVal(os.Getenv(args[0].AsString())), nil
		},
	})

	ctx := &hcl.EvalContext{
		Functions: map[string]function.Function{},
	}
	ctx.Functions["env"] = EnvFunc

	return ctx
}

func decodeBody(b *hclsyntax.Block, p interface{}) error {
	diag := gohcl.DecodeBody(b.Body, ctx, p)
	if diag.HasErrors() {
		return errors.New(diag.Error())
	}

	return nil
}
