package terraform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/jumppad-labs/hclconfig/convert"
	"github.com/jumppad-labs/hclconfig/resources"
	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	cclient "github.com/jumppad-labs/jumppad/pkg/clients/container"
	ctypes "github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	sdk "github.com/jumppad-labs/plugin-sdk"
	"github.com/kennygrant/sanitize"
	"github.com/zclconf/go-cty/cty"
)

const terraformImageName = "hashicorp/terraform"

var _ sdk.Provider = &TerraformProvider{}

// TerraformProvider provider allows the execution of terraform config
type TerraformProvider struct {
	config *Terraform
	client cclient.ContainerTasks
	log    sdk.Logger
}

func (p *TerraformProvider) Init(cfg htypes.Resource, l sdk.Logger) error {
	c, ok := cfg.(*Terraform)
	if !ok {
		return fmt.Errorf("unable to initialize Terraform provider, resource is not of type Terraform")
	}

	cli, err := clients.GenerateClients(l)
	if err != nil {
		return err
	}

	p.config = c
	p.client = cli.ContainerTasks
	p.log = l

	return nil
}

// Create a new terraform container
func (p *TerraformProvider) Create(ctx context.Context) error {
	if ctx.Err() != nil {
		p.log.Debug("context cancelled, skipping create", "ref", p.config.Meta.ID)
		return nil
	}

	p.log.Info("Creating Terraform", "ref", p.config.Meta.ID)

	err := p.generateVariables()
	if err != nil {
		return fmt.Errorf("unable to generate variables file: %w", err)
	}

	// terraform init & terraform apply
	id, err := p.createContainer()
	if err != nil {
		return fmt.Errorf("unable to create container for terraform.%s: %w", p.config.Meta.Name, err)
	}

	// always remove the container
	defer p.client.RemoveContainer(id, true)

	err = p.terraformApply(id)
	if err != nil {
		return fmt.Errorf("unable to run apply for terraform.%s: %w", p.config.Meta.Name, err)
	}

	err = p.generateOutput()
	if err != nil {
		return fmt.Errorf("unable to generate output: %w", err)
	}

	// set the checksum for the source folder
	hash, err := utils.HashDir(p.config.Source, "**/.terraform.lock.hcl")
	if err != nil {
		return fmt.Errorf("unable to hash source directory: %w", err)
	}

	p.config.SourceChecksum = hash

	return nil
}

// Destroy the terraform container
func (p *TerraformProvider) Destroy(ctx context.Context, force bool) error {
	if ctx.Err() != nil {
		p.log.Debug("context cancelled, skipping destroy", "ref", p.config.Meta.ID)
		return nil
	}

	// if force do not try to do a destroy, just exit
	if force {
		p.log.Info("Skipping Destroy Terraform", "ref", p.config.Meta.ID, "force", true)
		return nil
	}

	p.log.Info("Destroy Terraform", "ref", p.config.Meta.ID)

	id, err := p.createContainer()
	if err != nil {
		return fmt.Errorf("unable to create container for Terraform.%s: %w", p.config.Meta.Name, err)
	}

	// always remove the container
	defer p.client.RemoveContainer(id, true)

	err = p.terraformDestroy(id)
	if err != nil {
		return fmt.Errorf("unable to destroy Terraform configuration: %w", err)
	}

	// remove the temp folders
	os.RemoveAll(terraformStateFolder(p.config))

	return err
}

// Lookup satisfies the interface requirements but is not used
// as the resource is not persistent
func (p *TerraformProvider) Lookup() ([]string, error) {
	return []string{}, nil
}

func (p *TerraformProvider) Refresh(ctx context.Context) error {
	// has the source folder changed?
	changed, err := p.Changed()
	if err != nil {
		return err
	}

	if changed {
		// with Terraform resources we can just re-call apply rather than
		// destroying and then running create.
		p.log.Debug("Refresh Terraform", "ref", p.config.Meta.ID)
		return p.Create(ctx)
	}

	// nothing changed set the outputs as these are not persisted to state
	err = p.generateOutput()
	if err != nil {
		return fmt.Errorf("unable to set outputs: %w", err)
	}

	return nil
}

// Changed checks to see if the resource files have changed since the last apply
func (p *TerraformProvider) Changed() (bool, error) {
	// check if the hash for the source folder has changed
	newHash, err := utils.HashDir(p.config.Source, "**/.terraform.lock.hcl")
	if err != nil {
		return true, fmt.Errorf("error hashing source directory: %w", err)
	}

	if newHash != p.config.SourceChecksum {
		p.log.Debug("Terraform source folder changed", "ref", p.config.Meta.ID)
		return true, nil
	}

	p.log.Debug("Terraform source folder unchanged", "ref", p.config.Meta.ID)
	return false, nil
}

// generate tfvars file with the passed in variables
func (p *TerraformProvider) generateVariables() error {
	// do nothing if empty
	if p.config.Variables.IsNull() {
		return nil
	}

	statePath := terraformStateFolder(p.config)

	f := hclwrite.NewEmptyFile()
	root := f.Body()

	if !p.config.Variables.Type().IsObjectType() && !p.config.Variables.Type().IsMapType() && !p.config.Variables.Type().IsTupleType() {
		return fmt.Errorf("error: variables is not a map")
	}

	for k, v := range p.config.Variables.AsValueMap() {
		root.SetAttributeValue(k, v)
	}

	variablesPath := filepath.Join(statePath, "terraform.tfvars")
	err := os.WriteFile(variablesPath, f.Bytes(), 0755)
	if err != nil {
		return fmt.Errorf("unable to write variables to disk at %s", variablesPath)
	}

	return nil
}

func (p *TerraformProvider) createContainer() (string, error) {
	fqdn := utils.FQDN(p.config.Meta.Name, p.config.Meta.Module, p.config.Meta.Type)
	statePath := terraformStateFolder(p.config)
	cachePath := terraformCacheFolder()

	image := fmt.Sprintf("%s:%s", terraformImageName, p.config.Version)

	// set the plugin cache so this is re-used
	if p.config.Environment == nil {
		p.config.Environment = map[string]string{}
	}

	p.config.Environment["TF_PLUGIN_CACHE_DIR"] = "/var/lib/terraform.d"

	tf := ctypes.Container{
		Name:        fqdn,
		Image:       &ctypes.Image{Name: image},
		Environment: p.config.Environment,
	}

	for _, v := range p.config.Networks {
		tf.Networks = append(tf.Networks, ctypes.NetworkAttachment{
			ID:        v.ID,
			Name:      v.Name,
			IPAddress: v.IPAddress,
			Aliases:   v.Aliases,
		})
	}

	if len(tf.Networks) == 0 {
		tf.Networks = append(tf.Networks, ctypes.NetworkAttachment{
			ID:   "resource.network.main",
			Name: "main",
		})
	}

	// Add the config folder
	tf.Volumes = append(tf.Volumes, ctypes.Volume{
		Source:      p.config.Source,
		Destination: "/config",
		Type:        "bind",
		ReadOnly:    false,
	})

	// Add the state folder
	tf.Volumes = append(tf.Volumes, ctypes.Volume{
		Source:      statePath,
		Destination: "/var/lib/terraform",
		Type:        "bind",
		ReadOnly:    false,
	})

	// Add the plugin cache
	tf.Volumes = append(tf.Volumes, ctypes.Volume{
		Source:      cachePath,
		Destination: "/var/lib/terraform.d",
		Type:        "bind",
		ReadOnly:    false,
	})

	// Add any additional volumes
	for _, v := range p.config.Volumes {
		tf.Volumes = append(tf.Volumes, ctypes.Volume{
			Source:                      v.Source,
			Destination:                 v.Destination,
			Type:                        v.Type,
			ReadOnly:                    v.ReadOnly,
			BindPropagation:             v.BindPropagation,
			BindPropagationNonRecursive: v.BindPropagationNonRecursive,
			SelinuxRelabel:              v.SelinuxRelabel,
		})
	}

	tf.Entrypoint = []string{"tail"}
	tf.Command = []string{"-f", "/dev/null"} // ensure container does not immediately exit

	// pull any images needed for this container
	err := p.client.PullImage(*tf.Image, false)
	if err != nil {
		p.log.Error("Error pulling container image", "ref", p.config.Meta.ID, "image", tf.Image.Name)

		return "", err
	}

	id, err := p.client.CreateContainer(&tf)
	if err != nil {
		p.log.Error("Error creating container for terraform", "ref", p.config.Meta.Name, "image", tf.Image.Name, "networks", p.config.Networks)
		return "", err
	}

	return id, err
}

func (p *TerraformProvider) terraformApply(containerid string) error {

	// allways run the cleanup
	defer func() {
		script := "rm -rf /config/.terraform"
		_, err := p.client.ExecuteScript(containerid, script, []string{}, "/", "root", "", 300, nil)
		if err != nil {
			p.log.Debug("unable to remove .terraform folder", "error", err)
		}
	}()

	// build the environment variables
	envs := []string{}
	for k, v := range p.config.Environment {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}

	tfvarFlag := getTerraformVarsFlag(p.config)
	wd := path.Join("/config", p.config.WorkingDirectory)

	script := `#!/bin/sh
  terraform init \
    -no-color
  terraform apply \
    -no-color \
    -state=/var/lib/terraform/terraform.tfstate \
    -auto-approve`

	// add the tf vars flag if we have a file
	script = script + tfvarFlag

	script = script + `terraform output \
    -no-color \
    -state=/var/lib/terraform/terraform.tfstate \
    -json > /var/lib/terraform/output.json`

	planOutput := bytes.NewBufferString("")

	p.log.Debug("Running terraform apply", "id", p.config.Meta.ID, "script", script, "envs", envs, "wd", wd)

	_, err := p.client.ExecuteScript(containerid, script, envs, wd, "root", "", 300, planOutput)

	// write the plan output to the log
	p.config.ApplyOutput = planOutput.String()
	p.log.Debug("terraform apply output", "id", p.config.Meta.ID, "output", planOutput)

	if err != nil {
		p.log.Error("Error executing terraform apply", "ref", p.config.Meta.Name)
		err = fmt.Errorf("unable to execute terraform apply: %w", err)
		return err
	}

	return nil
}

func (p *TerraformProvider) generateOutput() error {
	statePath := terraformStateFolder(p.config)
	outputPath := path.Join(statePath, "output.json")

	data, err := os.ReadFile(outputPath)
	if err != nil {
		return fmt.Errorf("unable to read terraform output: %w", err)
	}

	var output map[string]interface{}
	err = json.Unmarshal(data, &output)
	if err != nil {
		return fmt.Errorf("unable to parse terraform output: %w", err)
	}

	values := map[string]cty.Value{}
	for k, v := range output {
		m, ok := v.(map[string]interface{})
		if !ok {
			return fmt.Errorf("terraform output is not in the correct format, expected map[string]interface{} for value but got %T", v)
		}

		value, err := convert.GoToCtyValue(m["value"])
		if err != nil {
			if reflect.TypeOf(m["type"]).Kind() == reflect.Slice {
				obj := map[string]cty.Value{}

				for l, w := range m["value"].(map[string]interface{}) {
					subvalue, err := convert.GoToCtyValue(w)
					if err != nil {
						p.log.Error("could not convert variable", "key", l, "value", w, "error", err)
						return err
					}

					obj[l] = subvalue
				}

				values[k] = cty.ObjectVal(obj)
			}
		} else {
			values[k] = value
		}
	}

	p.config.Output = cty.ObjectVal(values)

	return nil
}

func (p *TerraformProvider) terraformDestroy(containerid string) error {
	// build the environment variables
	envs := []string{}
	for k, v := range p.config.Environment {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}

	// check to see if the state files exist, if not then this resource might not
	// have been created correctly so just exit
	statePath := terraformStateFolder(p.config)
	_, err := os.Stat(path.Join(statePath, "terraform.tfstate"))
	if err != nil {
		return nil
	}

	_, err = os.Stat(path.Join(statePath, "terraform.tfvars"))
	if err != nil {
		return nil
	}

	wd := path.Join("/config", p.config.WorkingDirectory)

	tfvarFlag := getTerraformVarsFlag(p.config)

	script := `#!/bin/sh
  terraform init \
    -no-color
  terraform destroy \
    -no-color \
    -state=/var/lib/terraform/terraform.tfstate \
    -auto-approve`

	// add the tf vars flag if we have a file
	script = script + tfvarFlag

	p.log.Debug("Running terraform destroy", "id", p.config.Meta.ID, "script", script, "envs", envs, "wd", wd)

	_, err = p.client.ExecuteScript(containerid, script, envs, wd, "root", "", 300, p.log.StandardWriter())
	if err != nil {
		p.log.Error("Error executing terraform destroy", "ref", p.config.Meta.Name)
		err = fmt.Errorf("unable to execute terraform destroy: %w", err)
		return err
	}

	return nil
}

func getTerraformVarsFlag(r *Terraform) string {
	// do we have a vars file
	statePath := terraformStateFolder(r)
	tfvarFlag := ` \
    -var-file=/var/lib/terraform/terraform.tfvars
  `

	_, err := os.Stat(filepath.Join(statePath, "terraform.tfvars"))
	if err != nil {
		// vars file does not exit remove the flag
		return ""
	}

	return tfvarFlag
}

// GetTerraformFolder creates the terraform directory used by the application
func terraformStateFolder(r *Terraform) string {
	p := sanitize.Path(resources.FQRNFromResource(r).String())
	p = strings.Replace(p, ".", "_", -1)
	p = strings.Replace(p, "-", "_", -1)

	data := filepath.Join(utils.JumppadHome(), "terraform", "state", p)

	// create the folder if it does not exist
	os.MkdirAll(data, 0755)
	os.Chmod(data, 0755)

	return data
}

func terraformCacheFolder() string {
	// the cache folder is
	return utils.CacheFolder("terraform", 0755)
}
