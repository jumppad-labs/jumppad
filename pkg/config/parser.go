package config

// TODO how do we deal with multiple stanza with the same name

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gernest/front"
	"github.com/hashicorp/go-getter"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclparse"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"golang.org/x/xerrors"
)

// TODO this really needs to be a struct with configuration not
// separate methods.

var ctx *hcl.EvalContext

func init() {
	ctx = buildContext()
}

type ResourceTypeNotExistError struct {
	Type string
	File string
}

func (r ResourceTypeNotExistError) Error() string {
	return fmt.Sprintf("Resource type %s defined in file %s, does not exist. Please check the documentation for supported resources. We love PRs if you would like to create a resource of this type :)", r.Type, r.File)
}

// ParseFolder for Resource, Blueprint, and Variable files
// The onlyResources parameter allows you to specfiy that the parser
// only reads resource files and will ignore Blueprint and Varaible files.
// This is useful when recursively parsing such as when reading Modules
func ParseFolder(folder string, c *Config, onlyResources bool, variables map[string]string, variablesFile string) error {
	abs, _ := filepath.Abs(folder)

	// load the variables
	if !onlyResources {
		variableFiles, err := filepath.Glob(path.Join(abs, "*.vars"))
		if err != nil {
			return err
		}

		for _, f := range variableFiles {
			err := LoadValuesFile(f)
			if err != nil {
				return err
			}
		}

		// setup any variables which are passed as environment variables or in the collection
		SetVariables(variables)

		// load variables from any custom files
		if variablesFile != "" {
			err := LoadValuesFile(variablesFile)
			if err != nil {
				return err
			}
		}

		// pick up the blueprint file
		yardFilesHCL, err := filepath.Glob(path.Join(abs, "*.yard"))
		if err != nil {
			return err
		}

		yardFilesMD, err := filepath.Glob(path.Join(abs, "README.md"))
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
	}

	// Parse Resource files from the current folder
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

// LoadValuesFile loads variable values from a file
func LoadValuesFile(path string) error {
	parser := hclparse.NewParser()

	f, diag := parser.ParseHCLFile(path)
	if diag.HasErrors() {
		return errors.New(diag.Error())
	}

	attrs, _ := f.Body.JustAttributes()
	for name, attr := range attrs {
		val, _ := attr.Expr.Value(nil)

		setContextVariable(name, val)
	}

	return nil
}

func setContextVariable(key string, value interface{}) {
	var valMap map[string]cty.Value

	// get the existing map
	if m, ok := ctx.Variables["var"]; ok {
		valMap = m.AsValueMap()
	} else {
		valMap = map[string]cty.Value{}
	}

	switch v := value.(type) {
	case string:
		valMap[key] = cty.StringVal(v)
	case cty.Value:
		valMap[key] = v
	}

	ctx.Variables["var"] = cty.MapVal(valMap)
}

// SetVariables allow variables to be set from a collection or environment variables
// Precedence should be file, env, vars
func SetVariables(vars map[string]string) {
	// first any vars defined as environment variables
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "SY_VAR_") {
			parts := strings.Split(e, "=")
			setContextVariable(strings.Replace(parts[0], "SY_VAR_", "", -1), parts[1])
		}
	}

	// then set vars
	for k, v := range vars {
		setContextVariable(k, v)
	}
}

func parseYardHCL(file string, c *Config) error {
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
		fmt.Println("Error parsing README.md", err)
		return nil
	}

	bp := &Blueprint{}
	bp.HealthCheckTimeout = "30s"

	// set the default health check

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

	if a, ok := fr["health_check_timeout"].(string); ok {
		bp.HealthCheckTimeout = a
	}

	if a, ok := fr["shipyard_version"].(string); ok {
		bp.ShipyardVersion = a
	}

	if envs, ok := fr["env"].([]interface{}); ok {
		bp.Environment = []KV{}
		for _, e := range envs {
			parts := strings.Split(e.(string), "=")
			if len(parts) == 2 {
				bp.Environment = append(bp.Environment, KV{Key: parts[0], Value: parts[1]})
			}
		}
	}

	bp.Intro = body

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
		case string(TypeK8sCluster):
			cl := NewK8sCluster(b.Labels[0])

			err := decodeBody(file, b, cl)
			if err != nil {
				return err
			}

			// Process volumes
			// make sure mount paths are absolute
			for i, v := range cl.Volumes {
				cl.Volumes[i].Source = ensureAbsolute(v.Source, file)
			}

			c.AddResource(cl)

		case string(TypeOutput):
			cl := NewOutput(b.Labels[0])

			err := decodeBody(file, b, cl)
			if err != nil {
				return err
			}

			c.AddResource(cl)

		case string(TypeK8sConfig):
			h := NewK8sConfig(b.Labels[0])

			err := decodeBody(file, b, h)
			if err != nil {
				return err
			}

			// make all the paths absolute
			for i, p := range h.Paths {
				h.Paths[i] = ensureAbsolute(p, file)
			}

			c.AddResource(h)

		case string(TypeHelm):
			h := NewHelm(b.Labels[0])

			err := decodeBody(file, b, h)
			if err != nil {
				return err
			}

			// if ChartName is not set use the name of the chart use the name of the
			// resource
			if h.ChartName == "" {
				h.ChartName = h.Name
			}

			// only set absolute if is local folder
			if h.Chart != "" && utils.IsLocalFolder(ensureAbsolute(h.Chart, file)) {
				h.Chart = ensureAbsolute(h.Chart, file)
			}

			if h.Values != "" && utils.IsLocalFolder(ensureAbsolute(h.Values, file)) {
				h.Values = ensureAbsolute(h.Values, file)
			}

			c.AddResource(h)

		case string(TypeK8sIngress):
			i := NewK8sIngress(b.Labels[0])

			err := decodeBody(file, b, i)
			if err != nil {
				return err
			}

			c.AddResource(i)

		case string(TypeNomadCluster):
			cl := NewNomadCluster(b.Labels[0])

			err := decodeBody(file, b, cl)
			if err != nil {
				return err
			}

			// Process volumes
			// make sure mount paths are absolute
			for i, v := range cl.Volumes {
				cl.Volumes[i].Source = ensureAbsolute(v.Source, file)
			}

			c.AddResource(cl)

		case string(TypeNomadJob):
			h := NewNomadJob(b.Labels[0])

			err := decodeBody(file, b, h)
			if err != nil {
				return err
			}

			// make all the paths absolute
			for i, p := range h.Paths {
				h.Paths[i] = ensureAbsolute(p, file)
			}

			c.AddResource(h)

		case string(TypeNomadIngress):
			i := NewNomadIngress(b.Labels[0])

			err := decodeBody(file, b, i)
			if err != nil {
				return err
			}

			c.AddResource(i)

		case string(TypeNetwork):
			n := NewNetwork(b.Labels[0])

			err := decodeBody(file, b, n)
			if err != nil {
				return err
			}

			c.AddResource(n)

		case string(TypeIngress):
			i := NewIngress(b.Labels[0])

			err := decodeBody(file, b, i)
			if err != nil {
				return err
			}

			c.AddResource(i)

		case string(TypeContainer):
			co := NewContainer(b.Labels[0])

			err := decodeBody(file, b, co)
			if err != nil {
				return err
			}

			// process volumes
			for i, v := range co.Volumes {
				// make sure mount paths are absolute when type is bind
				if v.Type == "" || v.Type == "bind" {
					co.Volumes[i].Source = ensureAbsolute(v.Source, file)
				}
			}

			// make sure build paths are absolute
			if co.Build != nil {
				co.Build.Context = ensureAbsolute(co.Build.Context, file)
			}

			c.AddResource(co)

		case string(TypeContainerIngress):
			i := NewContainerIngress(b.Labels[0])

			err := decodeBody(file, b, i)
			if err != nil {
				return err
			}

			c.AddResource(i)

		case string(TypeSidecar):
			s := NewSidecar(b.Labels[0])

			err := decodeBody(file, b, s)
			if err != nil {
				return err
			}

			for i, v := range s.Volumes {
				s.Volumes[i].Source = ensureAbsolute(v.Source, file)
			}

			c.AddResource(s)

		case string(TypeDocs):
			do := NewDocs(b.Labels[0])

			err := decodeBody(file, b, do)
			if err != nil {
				return err
			}

			do.Path = ensureAbsolute(do.Path, file)

			c.AddResource(do)

		case string(TypeExecLocal):
			h := NewExecLocal(b.Labels[0])

			err := decodeBody(file, b, h)
			if err != nil {
				return err
			}

			c.AddResource(h)

		case string(TypeExecRemote):
			h := NewExecRemote(b.Labels[0])

			err := decodeBody(file, b, h)
			if err != nil {
				return err
			}

			// process volumes
			// make sure mount paths are absolute
			for i, v := range h.Volumes {
				h.Volumes[i].Source = ensureAbsolute(v.Source, file)
			}

			c.AddResource(h)

		case string(TypeModule):
			m := NewModule(b.Labels[0])

			err := decodeBody(file, b, m)
			if err != nil {
				return err
			}

			// import the source files for this module
			if !utils.IsLocalFolder(ensureAbsolute(m.Source, file)) {
				// get the details
				dst := utils.GetBlueprintLocalFolder(m.Source)
				err := getFiles(m.Source, dst)
				if err != nil {
					return err
				}

				// set the source to the local folder
				m.Source = dst
			}

			// set the absolute path
			m.Source = ensureAbsolute(m.Source, file)

			conf := New()
			// recursively parse references for the module
			// ensure we do load the values which might be in module folders
			err = ParseFolder(m.Source, conf, true, nil, "")
			if err != nil {
				return err
			}

			// add the parsed resources to the main but add the module name
			for _, r := range conf.Resources {
				r.Info().Module = m.Name
				r.Info().DependsOn = m.Depends
				c.AddResource(r)
			}

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

		case TypeContainerIngress:
			c := r.(*ContainerIngress)
			for _, n := range c.Networks {
				c.DependsOn = append(c.DependsOn, n.Name)
			}
			c.DependsOn = append(c.DependsOn, c.Target)
			c.DependsOn = append(c.DependsOn, c.Depends...)

		case TypeSidecar:
			c := r.(*Sidecar)
			c.DependsOn = append(c.DependsOn, c.Target)
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

		case TypeExecLocal:
			c := r.(*ExecLocal)
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

		case TypeHelm:
			c := r.(*Helm)
			c.DependsOn = append(c.DependsOn, c.Cluster)
			c.DependsOn = append(c.DependsOn, c.Depends...)

		case TypeK8sConfig:
			c := r.(*K8sConfig)
			c.DependsOn = append(c.DependsOn, c.Cluster)
			c.DependsOn = append(c.DependsOn, c.Depends...)

		case TypeK8sIngress:
			c := r.(*K8sIngress)
			for _, n := range c.Networks {
				c.DependsOn = append(c.DependsOn, n.Name)
			}
			c.DependsOn = append(c.DependsOn, c.Cluster)
			c.DependsOn = append(c.DependsOn, c.Depends...)

		case TypeNomadCluster:
			c := r.(*NomadCluster)
			for _, n := range c.Networks {
				c.DependsOn = append(c.DependsOn, n.Name)
			}
			c.DependsOn = append(c.DependsOn, c.Depends...)

		case TypeNomadIngress:
			c := r.(*NomadIngress)
			for _, n := range c.Networks {
				c.DependsOn = append(c.DependsOn, n.Name)
			}
			c.DependsOn = append(c.DependsOn, c.Cluster)
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

	var HomeFunc = function.New(&function.Spec{
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			return cty.StringVal(utils.HomeFolder()), nil
		},
	})

	var ShipyardFunc = function.New(&function.Spec{
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			return cty.StringVal(utils.ShipyardHome()), nil
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
			_, kcp, _ := utils.CreateKubeConfigPath(args[0].AsString())
			return cty.StringVal(kcp), nil
		},
	})

	var KubeConfigDockerFunc = function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name:             "k8s_config_docker",
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

	var FileFunc = function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name:             "path",
				Type:             cty.String,
				AllowDynamicType: true,
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			// get the current file path from the context
			path := ctx.Variables["path"].AsString()
			// conver the file path to an absolute
			fp := ensureAbsolute(args[0].AsString(), path)

			// read the contents of the file
			d, err := ioutil.ReadFile(fp)
			if err != nil {
				return cty.StringVal(""), err
			}

			return cty.StringVal(string(d)), nil
		},
	})

	var DataFunc = function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name:             "path",
				Type:             cty.String,
				AllowDynamicType: true,
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			return cty.StringVal(utils.GetDataFolder(args[0].AsString())), nil
		},
	})

	ctx := &hcl.EvalContext{
		Functions: map[string]function.Function{},
		Variables: map[string]cty.Value{},
	}

	ctx.Functions["env"] = EnvFunc
	ctx.Functions["k8s_config"] = KubeConfigFunc
	ctx.Functions["k8s_config_docker"] = KubeConfigDockerFunc
	ctx.Functions["home"] = HomeFunc
	ctx.Functions["shipyard"] = ShipyardFunc
	ctx.Functions["file"] = FileFunc
	ctx.Functions["data"] = DataFunc

	return ctx
}

func decodeBody(path string, b *hclsyntax.Block, p interface{}) error {
	// add the current file path to the context.
	// this allows any functions which require absolute paths to be able to
	// build them from relative paths.
	ctx.Variables["path"] = cty.StringVal(path)

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

func getFiles(source, dest string) error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// if the argument is a url fetch it first
	c := &getter.Client{
		Ctx:     context.Background(),
		Src:     source,
		Dst:     dest,
		Pwd:     pwd,
		Mode:    getter.ClientModeAny,
		Options: []getter.ClientOption{},
	}

	err = c.Get()
	if err != nil {
		return xerrors.Errorf("unable to fetch files from %s: %w", source, err)
	}

	return nil
}
