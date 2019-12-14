package config

// TODO how do we deal with multiple stanza with the same name

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclparse"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

var ctx *hcl.EvalContext

// ParseFolder for config entries
func ParseFolder(folder string, c *Config) error {
	ctx = buildContext()

	abs, _ := filepath.Abs(folder)

	// pick up the blueprint file
	yardFiles, err := filepath.Glob(path.Join(abs, "*.yard"))
	if err != nil {
		fmt.Println("err")
		return err
	}

	if len(yardFiles) > 0 {
		err := ParseYardFile(yardFiles[0], c)
		if err != nil {
			fmt.Println("err")
			return err
		}
	}

	// load files from the current folder
	files, err := filepath.Glob(path.Join(abs, "*.hcl"))
	if err != nil {
		fmt.Println("err")
		return err
	}

	// sub folders
	filesDir, err := filepath.Glob(path.Join(abs, "**/*.hcl"))
	if err != nil {
		fmt.Println("err")
		return err
	}

	files = append(files, filesDir...)

	for _, f := range files {
		err := ParseHCLFile(f, c)
		if err != nil {
			return err
		}
	}

	return nil
}

func ParseYardFile(file string, c *Config) error {
	parser := hclparse.NewParser()

	f, diag := parser.ParseHCLFile(file)
	if diag.HasErrors() {
		return errors.New(diag.Error())
	}

	body, ok := f.Body.(*hclsyntax.Body)
	if !ok {
		return errors.New("Error getting body")
	}

	bp := &Blueprint{}

	diag = gohcl.DecodeBody(body, ctx, bp)
	if diag.HasErrors() {
		return errors.New(diag.Error())
	}

	c.Blueprint = bp

	return nil
}

// ParseHCLFile parses a config file and adds it to the config
func ParseHCLFile(file string, c *Config) error {
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
			cl.Name = b.Labels[0]

			err := decodeBody(b, cl)
			if err != nil {
				return err
			}

			c.Clusters = append(c.Clusters, cl)
		case "network":
			if b.Labels[0] == "wan" {
				return ErrorWANExists
			}

			n := &Network{}
			n.Name = b.Labels[0]

			err := decodeBody(b, n)
			if err != nil {
				return err
			}

			c.Networks = append(c.Networks, n)
		case "helm":
			h := &Helm{}
			h.Name = b.Labels[0]

			err := decodeBody(b, h)
			if err != nil {
				return err
			}

			h.Chart = ensureAbsolute(h.Chart, file)
			h.Values = ensureAbsolute(h.Values, file)

			c.HelmCharts = append(c.HelmCharts, h)
		case "ingress":
			i := &Ingress{}
			i.Name = b.Labels[0]

			err := decodeBody(b, i)
			if err != nil {
				return err
			}

			c.Ingresses = append(c.Ingresses, i)
		case "container":
			co := &Container{}
			co.Name = b.Labels[0]

			err := decodeBody(b, co)
			if err != nil {
				return err
			}

			// process volumes
			// make sure mount paths are absolute
			for i, v := range co.Volumes {
				co.Volumes[i].Source = ensureAbsolute(v.Source, file)
			}

			c.Containers = append(c.Containers, co)

		case "docs":
			do := &Docs{}
			do.Name = b.Labels[0]

			err := decodeBody(b, do)
			if err != nil {
				return err
			}

			do.Path = ensureAbsolute(do.Path, file)

			c.Docs = do
		}
	}

	return nil
}

// ParseReferences links the object references in config elements
func ParseReferences(c *Config) error {
	// link the networks in the clusters
	for _, cl := range c.Clusters {
		cl.WANRef = c.WAN
		cl.NetworkRef = findNetworkRef(cl.Network, c)
	}

	for _, co := range c.Containers {
		co.WANRef = c.WAN
		co.NetworkRef = findNetworkRef(co.Network, c)
	}

	for _, hc := range c.HelmCharts {
		hc.ClusterRef = findClusterRef(hc.Cluster, c)
	}

	for _, in := range c.Ingresses {
		in.WANRef = c.WAN
		in.TargetRef = findTargetRef(in.Target, c)

		if c, ok := in.TargetRef.(*Cluster); ok {
			in.NetworkRef = c.NetworkRef
		} else {
			in.NetworkRef = in.TargetRef.(*Container).NetworkRef
		}
	}

	if c.Docs != nil {
		c.Docs.WANRef = c.WAN
	}

	return nil
}

func findNetworkRef(name string, c *Config) *Network {
	nn := strings.Split(name, ".")[1]

	for _, n := range c.Networks {
		if n.Name == nn {
			return n
		}
	}

	return nil
}

func findClusterRef(name string, c *Config) *Cluster {
	nn := strings.Split(name, ".")[1]

	for _, c := range c.Clusters {
		if c.Name == nn {
			return c
		}
	}

	return nil
}

func findContainerRef(name string, c *Config) *Container {
	nn := strings.Split(name, ".")[1]

	for _, c := range c.Containers {
		if c.Name == nn {
			return c
		}
	}

	return nil
}

func findTargetRef(name string, c *Config) interface{} {
	// target can be either a cluster or a container
	cl := findClusterRef(name, c)
	if cl != nil {
		return cl
	}

	co := findContainerRef(name, c)
	if co != nil {
		return co
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

// ensureAbsolute ensure that the given path is either absolute or
// if relative is converted to abasolute based on the path of the config
func ensureAbsolute(path, file string) string {
	if filepath.IsAbs(path) {
		return path
	}

	// path is relative so make absolute using the current file path as base
	baseDir := filepath.Dir(file)
	return filepath.Join(baseDir, path)
}
