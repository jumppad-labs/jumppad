package terraform

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2/hclwrite"
	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	cclient "github.com/jumppad-labs/jumppad/pkg/clients/container"
	"github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	ctypes "github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"golang.org/x/xerrors"
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

	// generate tfvars file with the passed in variables
	f := hclwrite.NewEmptyFile()
	root := f.Body()

	variables := p.config.Variables.AsValueMap()
	for k, v := range variables {
		root.SetAttributeValue(k, v)
	}

	variablesPath := filepath.Join(terraformPath, "terraform.tfvars")
	err := os.WriteFile(variablesPath, f.Bytes(), 0755)
	if err != nil {
		return fmt.Errorf("unable to write variables to disk at %s", variablesPath)
	}

	// terraform init & terraform apply
	id, err := p.createContainer(terraformPath)
	if err != nil {
		return xerrors.Errorf("unable to create container for terraform.%s: %w", p.config.Name, err)
	}

	err = p.terraformApply(id)
	p.client.RemoveContainer(id, true)

	return err
}

// Destroy the terraform container
func (p *TerraformProvider) Destroy() error {
	p.log.Info("Destroy Terraform", "ref", p.config.ID)

	terraformPath := utils.GetTerraformFolder(p.config.Name, 0775)

	id, err := p.createContainer(terraformPath)
	if err != nil {
		return xerrors.Errorf("unable to create container for terraform.%s: %w", p.config.Name, err)
	}

	err = p.terraformDestroy(id)
	p.client.RemoveContainer(id, true)

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

	// set state directory to jumppad home dir
	terraformPath := utils.GetTerraformFolder(p.config.Name, 0775)

	// generate tfvars file with the passed in variables
	f := hclwrite.NewEmptyFile()
	root := f.Body()

	variables := p.config.Variables.AsValueMap()
	for k, v := range variables {
		root.SetAttributeValue(k, v)
	}

	variablesPath := filepath.Join(terraformPath, "terraform.tfvars")
	err := os.WriteFile(variablesPath, f.Bytes(), 0755)
	if err != nil {
		return fmt.Errorf("unable to write variables to disk at %s", variablesPath)
	}

	// terraform init & terraform apply
	id, err := p.createContainer(terraformPath)
	if err != nil {
		return xerrors.Errorf("unable to create container for terraform.%s: %w", p.config.Name, err)
	}

	err = p.terraformApply(id)
	p.client.RemoveContainer(id, true)

	return err
}

func (p *TerraformProvider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.Name)

	return false, nil
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
		tf.Networks = append(tf.Networks, types.NetworkAttachment{
			ID:        v.ID,
			Name:      v.Name,
			IPAddress: v.IPAddress,
			Aliases:   v.Aliases,
		})
	}

	for _, v := range p.config.Volumes {
		tf.Volumes = append(tf.Volumes, types.Volume{
			Source:                      v.Source,
			Destination:                 v.Destination,
			Type:                        v.Type,
			ReadOnly:                    v.ReadOnly,
			BindPropagation:             v.BindPropagation,
			BindPropagationNonRecursive: v.BindPropagationNonRecursive,
		})
	}

	tf.Volumes = append(tf.Volumes, types.Volume{
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
	`

	_, err := p.client.ExecuteScript(id, script, envs, p.config.WorkingDirectory, "root", "", 300, p.log.StandardWriter())
	if err != nil {
		p.log.Error("Error executing terraform apply", "ref", p.config.Name)
		err = fmt.Errorf("Unable to execute terraform apply: %w", err)
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
	terraform destroy \
		-state=/var/lib/terraform/terraform.tfstate \
		-var-file=/var/lib/terraform/terraform.tfvars \
		-auto-approve
	`
	_, err := p.client.ExecuteScript(id, script, envs, p.config.WorkingDirectory, "root", "", 300, p.log.StandardWriter())
	if err != nil {
		p.log.Error("Error executing terraform destroy", "ref", p.config.Name)
		err = fmt.Errorf("Unable to execute terraform destroy: %w", err)
		return err
	}

	return nil
}
