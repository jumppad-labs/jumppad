package config

// TODO how do we deal with multiple stanza with the same name

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gernest/front"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclparse"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

var ctx *hcl.EvalContext

type ResourceTypeNotExistError struct {
	Type string
	File string
}

func (r ResourceTypeNotExistError) Error() string {
	return fmt.Sprintf("Resource type %s defined in file %s, does not exist. Please check the documentation for supported resources. We love PRs if you would like to create a resource of this type :)", r.Type, r.File)
}

// ParseFolder for config entries
func ParseFolder(folder string, c *Config) error {
	ctx = buildContext()

	abs, _ := filepath.Abs(folder)

	// pick up the blueprint file
	yardFilesHCL, err := filepath.Glob(path.Join(abs, "*.yard"))
	if err != nil {
		return err
	}

	yardFilesMD, err := filepath.Glob(path.Join(abs, "*.md"))
	if err != nil {
		return err
	}

	yardFiles := []string{}
	yardFiles = append(yardFiles, yardFilesHCL...)
	yardFiles = append(yardFiles, yardFilesMD...)

	if len(yardFiles) > 0 {
		err := ParseYardFile(yardFiles[0], c)
		if err != nil {
			return err
		}
	}

	// load files from the current folder
	files, err := filepath.Glob(path.Join(abs, "*.hcl"))
	if err != nil {
		return err
	}

	for _, f := range files {
		err := ParseHCLFile(f, c)
		if err != nil {
			return err
		}
	}

	return nil
}

// ParseYardFile parses a blueprint configuration file
func ParseYardFile(file string, c *Config) error {
	if filepath.Ext(file) == ".yard" {
		return parseYardHCL(file, c)
	}

	return parseYardMarkdown(file, c)
}

func parseYardHCL(file string, c *Config) error {
	ctx = buildContext()

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

// parseYardMarkdown extracts the blueprint information from the frontmatter
// when a blueprint file is of type markdown
func parseYardMarkdown(file string, c *Config) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}

	m := front.NewMatter()
	m.Handle("---", front.YAMLHandler)

	fr, body, err := m.Parse(f)
	if err != nil && err != front.ErrIsEmpty {
		return err
	}

	bp := &Blueprint{}

	if a, ok := fr["author"].(string); ok {
		bp.Author = a
	}

	if a, ok := fr["title"].(string); ok {
		bp.Title = a
	}

	if a, ok := fr["slug"].(string); ok {
		bp.Slug = a
	}

	if a, ok := fr["browser_windows"].(string); ok {
		bp.BrowserWindows = strings.Split(a, ",")
	}

	bp.Intro = body

	c.Blueprint = bp
	return nil
}

// ParseHCLFile parses a config file and adds it to the config
func ParseHCLFile(file string, c *Config) error {
	ctx = buildContext()
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
		case string(TypeK8sCluster):
			cl := NewK8sCluster(b.Labels[0])

			err := decodeBody(b, cl)
			if err != nil {
				return err
			}

			c.AddResource(cl)

		case string(TypeK8sConfig):
			h := NewK8sConfig(b.Labels[0])

			err := decodeBody(b, h)
			if err != nil {
				return err
			}

			// make all the paths absolute
			for i, p := range h.Paths {
				h.Paths[i] = ensureAbsolute(p, file)
			}

			c.AddResource(h)

		case string(TypeNomadCluster):
			cl := NewNomadCluster(b.Labels[0])

			err := decodeBody(b, cl)
			if err != nil {
				return err
			}

			c.AddResource(cl)

		case string(TypeNomadJob):
			h := NewNomadJob(b.Labels[0])

			err := decodeBody(b, h)
			if err != nil {
				return err
			}

			// make all the paths absolute
			for i, p := range h.Paths {
				h.Paths[i] = ensureAbsolute(p, file)
			}

			c.AddResource(h)

		case string(TypeNetwork):
			n := NewNetwork(b.Labels[0])

			err := decodeBody(b, n)
			if err != nil {
				return err
			}

			c.AddResource(n)

		case string(TypeHelm):
			h := NewHelm(b.Labels[0])

			err := decodeBody(b, h)
			if err != nil {
				return err
			}

			h.Chart = ensureAbsolute(h.Chart, file)
			h.Values = ensureAbsolute(h.Values, file)

			c.AddResource(h)

		case string(TypeIngress):
			i := NewIngress(b.Labels[0])

			err := decodeBody(b, i)
			if err != nil {
				return err
			}

			c.AddResource(i)

		case string(TypeContainer):
			co := NewContainer(b.Labels[0])

			err := decodeBody(b, co)
			if err != nil {
				return err
			}

			// process volumes
			// make sure mount paths are absolute
			for i, v := range co.Volumes {
				co.Volumes[i].Source = ensureAbsolute(v.Source, file)
			}

			c.AddResource(co)

		case string(TypeDocs):
			do := NewDocs(b.Labels[0])

			err := decodeBody(b, do)
			if err != nil {
				return err
			}

			do.Path = ensureAbsolute(do.Path, file)

			c.AddResource(do)

		case string(TypeExecLocal):
			h := NewExecLocal(b.Labels[0])

			err := decodeBody(b, h)
			if err != nil {
				return err
			}

			h.Script = ensureAbsolute(h.Script, file)

			c.AddResource(h)

		case string(TypeExecRemote):
			h := NewExecRemote(b.Labels[0])

			err := decodeBody(b, h)
			if err != nil {
				return err
			}

			if h.Script != "" {
				h.Script = ensureAbsolute(h.Script, file)
			}

			// process volumes
			// make sure mount paths are absolute
			for i, v := range h.Volumes {
				h.Volumes[i].Source = ensureAbsolute(v.Source, file)
			}

			c.AddResource(h)
		default:
			return ResourceTypeNotExistError{string(b.Type), file}
		}
	}

	return nil
}

// ParseReferences links the object references in config elements
func ParseReferences(c *Config) error {
	for _, r := range c.Resources {
		switch r.Info().Type {
		case TypeContainer:
			c := r.(*Container)
			for _, n := range c.Networks {
				c.DependsOn = append(c.DependsOn, n.Name)
			}
			c.DependsOn = append(c.DependsOn, c.Depends...)
		case TypeDocs:
			c := r.(*Docs)
			for _, n := range c.Networks {
				c.DependsOn = append(c.DependsOn, n.Name)
			}
			c.DependsOn = append(c.DependsOn, c.Depends...)
		case TypeExecRemote:
			c := r.(*ExecRemote)
			for _, n := range c.Networks {
				c.DependsOn = append(c.DependsOn, n.Name)
			}

			c.DependsOn = append(c.DependsOn, c.Depends...)

			// target is optional
			if c.Target != "" {
				c.DependsOn = append(c.DependsOn, c.Target)
			}
		case TypeHelm:
			c := r.(*Helm)
			c.DependsOn = append(c.DependsOn, c.Cluster)
			c.DependsOn = append(c.DependsOn, c.Depends...)

		case TypeK8sConfig:
			c := r.(*K8sConfig)
			c.DependsOn = append(c.DependsOn, c.Cluster)
			c.DependsOn = append(c.DependsOn, c.Depends...)

		case TypeIngress:
			c := r.(*Ingress)
			for _, n := range c.Networks {
				c.DependsOn = append(c.DependsOn, n.Name)
			}
			c.DependsOn = append(c.DependsOn, c.Target)
			c.DependsOn = append(c.DependsOn, c.Depends...)
		case TypeK8sCluster:
			c := r.(*K8sCluster)
			for _, n := range c.Networks {
				c.DependsOn = append(c.DependsOn, n.Name)
			}
			c.DependsOn = append(c.DependsOn, c.Depends...)
		case TypeNomadCluster:
			c := r.(*NomadCluster)
			for _, n := range c.Networks {
				c.DependsOn = append(c.DependsOn, n.Name)
			}
			c.DependsOn = append(c.DependsOn, c.Depends...)
		case TypeNomadJob:
			c := r.(*NomadJob)
			c.DependsOn = append(c.DependsOn, c.Cluster)
			c.DependsOn = append(c.DependsOn, c.Depends...)
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

	var KubeConfigFunc = function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name:             "k8s_config",
				Type:             cty.String,
				AllowDynamicType: true,
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			_, _, kcp := utils.CreateKubeConfigPath(args[0].AsString())
			return cty.StringVal(kcp), nil
		},
	})

	ctx := &hcl.EvalContext{
		Functions: map[string]function.Function{},
	}
	ctx.Functions["env"] = EnvFunc
	ctx.Functions["k8s_config"] = KubeConfigFunc

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
	file, _ = filepath.Abs(file)
	baseDir := filepath.Dir(file)
	return filepath.Join(baseDir, path)
}
