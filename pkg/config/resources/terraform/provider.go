package terraform

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/jumppad-labs/hclconfig/convert"
	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	cclient "github.com/jumppad-labs/jumppad/pkg/clients/container"
	ctypes "github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/zclconf/go-cty/cty"
)

const terraformImageName = "hashicorp/terraform"
const terraformVersion = "1.5"

// TerraformProvider provider allows the execution of terraform config
type TerraformProvider struct {
	config *Terraform
	client cclient.ContainerTasks
	log    logger.Logger
}

func (p *TerraformProvider) Init(cfg htypes.Resource, l logger.Logger) error {
	c, ok := cfg.(*Terraform)
	if !ok {
		return fmt.Errorf("unable to initialize ImageCache provider, resource is not of type ImageCache")
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

	// set state directory to jumppad home dir
	terraformPath := utils.GetTerraformFolder(p.config.Name, 0775)

	err := p.generateVariables(terraformPath)
	if err != nil {
		return fmt.Errorf("unable to generate variables file: %w", err)
	}

	// terraform init & terraform apply
	id, err := p.createContainer(terraformPath)
	if err != nil {
		return fmt.Errorf("unable to create container for terraform.%s: %w", p.config.Name, err)
	}

	err = p.terraformApply(id)
	p.client.RemoveContainer(id, true)

	if err != nil {
		return fmt.Errorf("unable to apply terraform configuration: %w", err)
	}

	outputPath := filepath.Join(terraformPath, "output.json")
	err = p.generateOutput(outputPath)

	return err
}

// Destroy the terraform container
func (p *TerraformProvider) Destroy() error {
	p.log.Info("Destroy Terraform", "ref", p.config.ID)

	terraformPath := utils.GetTerraformFolder(p.config.Name, 0775)

	id, err := p.createContainer(terraformPath)
	if err != nil {
		return fmt.Errorf("unable to create container for terraform.%s: %w", p.config.Name, err)
	}

	err = p.terraformDestroy(id)
	p.client.RemoveContainer(id, true)

	if err != nil {
		return fmt.Errorf("unable to destroy terraform configuration: %w", err)
	}

	// clean up the terraform folder
	err = os.RemoveAll(utils.GetTerraformFolder("", os.ModePerm))

	return err
}

// Lookup satisfies the interface requirements but is not used
// as the resource is not persistent
func (p *TerraformProvider) Lookup() ([]string, error) {
	return []string{}, nil
}

func (p *TerraformProvider) Refresh() error {
	p.log.Debug("Refresh Terraform", "ref", p.config.ID)
	return p.Create()
}

func (p *TerraformProvider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.Name)
	return false, nil
}

// generate tfvars file with the passed in variables
func (p *TerraformProvider) generateVariables(path string) error {
	f := hclwrite.NewEmptyFile()
	root := f.Body()

	variables, diag := p.config.Variables.(*hcl.Attribute).Expr.Value(nil)
	if diag.HasErrors() {
		return fmt.Errorf(diag.Error())
	}

	for k, v := range variables.AsValueMap() {
		root.SetAttributeValue(k, v)
	}

	variablesPath := filepath.Join(path, "terraform.tfvars")
	err := os.WriteFile(variablesPath, f.Bytes(), 0755)
	if err != nil {
		return fmt.Errorf("unable to write variables to disk at %s", variablesPath)
	}

	return nil
}

func (p *TerraformProvider) generateOutput(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("unable to read terraform output: %w", err)
	}

	var output map[string]interface{}
	err = json.Unmarshal(data, &output)
	if err != nil {
		return fmt.Errorf("unable to parse terraform output: %w", err)
	}

	// p.config.Output = output

	p.log.Warn("output", "interface", output)

	values := map[string]cty.Value{}
	for k, v := range output {
		m := v.(map[string]interface{})
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

	p.log.Warn("output", "cty", cty.ObjectVal(values))

	p.config.Output = cty.ObjectVal(values)

	return nil
}

func (p *TerraformProvider) createContainer(path string) (string, error) {
	fqdn := utils.FQDN(p.config.Name, p.config.Module, p.config.Type)

	image := fmt.Sprintf("%s:%s", terraformImageName, terraformVersion)

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

	for _, v := range p.config.Volumes {
		tf.Volumes = append(tf.Volumes, ctypes.Volume{
			Source:                      v.Source,
			Destination:                 v.Destination,
			Type:                        v.Type,
			ReadOnly:                    v.ReadOnly,
			BindPropagation:             v.BindPropagation,
			BindPropagationNonRecursive: v.BindPropagationNonRecursive,
		})
	}

	tf.Volumes = append(tf.Volumes, ctypes.Volume{
		Source:      path,
		Destination: "/var/lib/terraform",
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
		p.log.Error("Error creating container for terraform", "ref", p.config.Name, "image", tf.Image.Name, "networks", p.config.Networks, "volumes", p.config.Volumes)
		return "", err
	}

	return id, err
}

func (p *TerraformProvider) terraformApply(id string) error {
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
		-json > /var/lib/terraform/output.json
	`

	_, err := p.client.ExecuteScript(id, script, envs, p.config.WorkingDirectory, "root", "", 300, p.log.StandardWriter())
	if err != nil {
		p.log.Error("Error executing terraform apply", "ref", p.config.Name)
		err = fmt.Errorf("unable to execute terraform apply: %w", err)
		return err
	}

	return nil
}

func (p *TerraformProvider) terraformDestroy(id string) error {
	// build the environment variables
	envs := []string{}
	for k, v := range p.config.Environment {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}

	script := `#!/bin/sh
	terraform init
	terraform destroy \
		-state=/var/lib/terraform/terraform.tfstate \
		-var-file=/var/lib/terraform/terraform.tfvars \
		-auto-approve
	`
	_, err := p.client.ExecuteScript(id, script, envs, p.config.WorkingDirectory, "root", "", 300, p.log.StandardWriter())
	if err != nil {
		p.log.Error("Error executing terraform destroy", "ref", p.config.Name)
		err = fmt.Errorf("unable to execute terraform destroy: %w", err)
		return err
	}

	return nil
}
