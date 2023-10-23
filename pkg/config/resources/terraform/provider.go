package terraform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/jumppad-labs/hclconfig/convert"
	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	cclient "github.com/jumppad-labs/jumppad/pkg/clients/container"
	ctypes "github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/kennygrant/sanitize"
	"github.com/zclconf/go-cty/cty"
)

const terraformImageName = "hashicorp/terraform"

// TerraformProvider provider allows the execution of terraform config
type TerraformProvider struct {
	config *Terraform
	client cclient.ContainerTasks
	log    logger.Logger
}

func (p *TerraformProvider) Init(cfg htypes.Resource, l logger.Logger) error {
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
func (p *TerraformProvider) Create() error {
	p.log.Info("Creating Terraform", "ref", p.config.ID)

	err := p.generateVariables()
	if err != nil {
		return fmt.Errorf("unable to generate variables file: %w", err)
	}

	// terraform init & terraform apply
	id, err := p.createContainer()
	if err != nil {
		return fmt.Errorf("unable to create container for terraform.%s: %w", p.config.Name, err)
	}

	// always remove the container
	defer p.client.RemoveContainer(id, true)

	err = p.terraformApply(id)
	if err != nil {
		return fmt.Errorf("unable to run apply for terraform.%s: %w", p.config.Name, err)
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
func (p *TerraformProvider) Destroy() error {
	p.log.Info("Destroy Terraform", "ref", p.config.ID)

	id, err := p.createContainer()
	if err != nil {
		return fmt.Errorf("unable to create container for Terraform.%s: %w", p.config.Name, err)
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

func (p *TerraformProvider) Refresh() error {
	// has the source folder changed?
	changed, err := p.Changed()
	if err != nil {
		return err
	}

	if changed {
		// with Terraform resources we can just re-call apply rather than
		// destroying and then running create.
		p.log.Debug("Refresh Terraform", "ref", p.config.ID)
		return p.Create()
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
		p.log.Debug("Terraform source folder changed", "ref", p.config.ID)
		return true, nil
	}

	p.log.Debug("Terraform source folder unchanged", "ref", p.config.ID)
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
	fqdn := utils.FQDN(p.config.Name, p.config.Module, p.config.Type)
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
	})

	// Add the plugin cache
	tf.Volumes = append(tf.Volumes, ctypes.Volume{
		Source:      cachePath,
		Destination: "/var/lib/terraform.d",
	})

	tf.Entrypoint = []string{}
	tf.Command = []string{"tail", "-f", "/dev/null"} // ensure container does not immediately exit

	// pull any images needed for this container
	err := p.client.PullImage(*tf.Image, false)
	if err != nil {
		p.log.Error("Error pulling container image", "ref", p.config.ID, "image", tf.Image.Name)

		return "", err
	}

	id, err := p.client.CreateContainer(&tf)
	if err != nil {
		p.log.Error("Error creating container for terraform", "ref", p.config.Name, "image", tf.Image.Name, "networks", p.config.Networks)
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

	script := `#!/bin/sh
	terraform init
	terraform apply \
		-state=/var/lib/terraform/terraform.tfstate \
		-var-file=/var/lib/terraform/terraform.tfvars \
		-auto-approve
	terraform output \
		-state=/var/lib/terraform/terraform.tfstate \
		-json > /var/lib/terraform/output.json`

	wd := path.Join("/config", p.config.WorkingDirectory)

	planOutput := bytes.NewBufferString("")

	_, err := p.client.ExecuteScript(containerid, script, envs, wd, "root", "", 300, planOutput)

	// write the plan output to the log
	p.config.ApplyOutput = planOutput.String()
	p.log.Debug("terraform apply output", "id", p.config.ID, "output", planOutput)

	if err != nil {
		p.log.Error("Error executing terraform apply", "ref", p.config.Name)
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

	script := `#!/bin/sh
	terraform init
	terraform destroy \
		-state=/var/lib/terraform/terraform.tfstate \
		-var-file=/var/lib/terraform/terraform.tfvars \
		-auto-approve
	`
	_, err = p.client.ExecuteScript(containerid, script, envs, wd, "root", "", 300, p.log.StandardWriter())
	if err != nil {
		p.log.Error("Error executing terraform destroy", "ref", p.config.Name)
		err = fmt.Errorf("unable to execute terraform destroy: %w", err)
		return err
	}

	return nil
}

// GetTerraformFolder creates the terraform directory used by the application
func terraformStateFolder(r *Terraform) string {
	p := sanitize.Path(htypes.FQDNFromResource(r).String())
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
