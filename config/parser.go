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

// Config defines the stack config
type Config struct {
	Clusters   []*Cluster
	Containers []*Container
	Networks   []*Network
	HelmCharts []*Helm
	Ingresses  []*Ingress
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
		case "network":
			n := &Network{}
			n.name = b.Labels[0]

			err := decodeBody(b, n)
			if err != nil {
				return err
			}

			c.Networks = append(c.Networks, n)
		case "helm":
			h := &Helm{}
			h.name = b.Labels[0]

			err := decodeBody(b, h)
			if err != nil {
				return err
			}

			c.HelmCharts = append(c.HelmCharts, h)
		case "ingress":
			i := &Ingress{}
			i.name = b.Labels[0]

			err := decodeBody(b, i)
			if err != nil {
				return err
			}

			c.Ingresses = append(c.Ingresses, i)
		case "container":
			co := &Container{}
			co.name = b.Labels[0]

			err := decodeBody(b, co)
			if err != nil {
				return err
			}

			c.Containers = append(c.Containers, co)
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

func ParseReferences(c *Config) error {
	// link the networks in the clusters
	for _, cl := range c.Clusters {
		cl.networkRef = findNetworkRef(cl.Network, c)
	}

	for _, cl := range c.Containers {
		cl.networkRef = findNetworkRef(cl.Network, c)
	}

	for _, hc := range c.HelmCharts {
		hc.clusterRef = findClusterRef(hc.Cluster, c)
	}

	for _, hc := range c.Ingresses {
		hc.targetRef = findTargetRef(hc.Target, c)
	}

	return nil
}

func findNetworkRef(name string, c *Config) *Network {
	nn := strings.Split(name, ".")[1]

	for _, n := range c.Networks {
		if n.name == nn {
			return n
		}
	}

	return nil
}

func findClusterRef(name string, c *Config) *Cluster {
	nn := strings.Split(name, ".")[1]

	for _, c := range c.Clusters {
		if c.name == nn {
			return c
		}
	}

	return nil
}

func findContainerRef(name string, c *Config) *Container {
	nn := strings.Split(name, ".")[1]

	for _, c := range c.Containers {
		if c.name == nn {
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

func generateOrder(c *Config) []interface{} {
	oc := make([]interface{}, 0)

	// first elements to create are networks
	for _, n := range c.Networks {
		oc = append(oc, n)
	}

	for _, c := range c.Containers {
		oc = append(oc, c)
	}

	return oc
}
