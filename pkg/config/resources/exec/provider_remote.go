package exec

import (
	"fmt"

	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	cclient "github.com/jumppad-labs/jumppad/pkg/clients/container"
	"github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	ctypes "github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	sdk "github.com/jumppad-labs/plugin-sdk"
	"golang.org/x/xerrors"
)

// ExecRemote provider allows the execution of arbitrary commands on an existing target or
// can create a new container before running
type RemoteProvider struct {
	config *RemoteExec
	client cclient.ContainerTasks
	log    sdk.Logger
}

func (p *RemoteProvider) Init(cfg htypes.Resource, l sdk.Logger) error {
	c, ok := cfg.(*RemoteExec)
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

// Create a new execution instance
func (p *RemoteProvider) Create() error {
	p.log.Info("Remote executing script", "ref", p.config.ID)
	p.log.Warn("This resource is deprecated and will be removed in a future version of Jumppad, please use exec instead")

	// execution target id
	targetID := ""

	if p.config.Target == nil {
		// Not using existing target create new container
		id, err := p.createRemoteExecContainer()
		if err != nil {
			return xerrors.Errorf("unable to create container for exec_remote.%s: %w", p.config.Name, err)
		}

		targetID = id
	} else {
		ids, err := p.client.FindContainerIDs(p.config.Target.ContainerName)
		if err != nil {
			return xerrors.Errorf("unable to find remote exec target: %w", err)
		}

		if len(ids) != 1 {
			return xerrors.Errorf("unable to find remote exec target %s", p.config.Target)
		}

		targetID = ids[0]
	}

	// execute the script in the container
	script := p.config.Script

	// build the environment variables
	envs := []string{}

	for k, v := range p.config.Environment {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}

	user := ""
	group := ""

	if p.config.RunAs != nil {
		user = p.config.RunAs.User
		group = p.config.RunAs.Group
	}

	_, err := p.client.ExecuteScript(targetID, script, envs, p.config.WorkingDirectory, user, group, 300, p.log.StandardWriter())
	if err != nil {
		p.log.Error("Error executing command", "ref", p.config.Name, "image", p.config.Image, "script", p.config.Script)
		err = xerrors.Errorf("Unable to execute command: in remote container: %w", err)
	}

	// destroy the container if we created one
	if p.config.Target == nil {
		p.client.RemoveContainer(targetID, true)
	}

	return err
}

// Destroy satisfies the interface requirements but is not used as the
// resource is not persistent
func (p *RemoteProvider) Destroy() error {
	return nil
}

// Lookup satisfies the interface requirements but is not used
// as the resource is not persistent
func (p *RemoteProvider) Lookup() ([]string, error) {
	return []string{}, nil
}

func (p *RemoteProvider) Refresh() error {
	p.log.Debug("Refresh Remote Exec", "ref", p.config.ID)

	return nil
}

func (p *RemoteProvider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.Name)

	return false, nil
}

func (p *RemoteProvider) createRemoteExecContainer() (string, error) {
	fqdn := utils.FQDN(p.config.Name, p.config.Module, p.config.Type)

	new := ctypes.Container{
		Name:        fqdn,
		Image:       &ctypes.Image{Name: p.config.Image.Name, Username: p.config.Image.Username, Password: p.config.Image.Password},
		Environment: p.config.Environment,
	}

	for _, v := range p.config.Networks {
		new.Networks = append(new.Networks, types.NetworkAttachment{
			ID:        v.ID,
			Name:      v.Name,
			IPAddress: v.IPAddress,
			Aliases:   v.Aliases,
		})
	}

	for _, v := range p.config.Volumes {
		new.Volumes = append(new.Volumes, types.Volume{
			Source:                      v.Source,
			Destination:                 v.Destination,
			Type:                        v.Type,
			ReadOnly:                    v.ReadOnly,
			BindPropagation:             v.BindPropagation,
			BindPropagationNonRecursive: v.BindPropagationNonRecursive,
			SelinuxRelabel:              v.SelinuxRelabel,
		})
	}

	new.Entrypoint = []string{}
	new.Command = []string{"tail", "-f", "/dev/null"} // ensure container does not immediately exit

	// pull any images needed for this container
	err := p.client.PullImage(*new.Image, false)
	if err != nil {
		p.log.Error("Error pulling container image", "ref", p.config.ID, "image", new.Image.Name)

		return "", err
	}

	id, err := p.client.CreateContainer(&new)
	if err != nil {
		p.log.Error("Error creating container for remote exec", "ref", p.config.Name, "image", p.config.Image, "networks", p.config.Networks, "volumes", p.config.Volumes)
		return "", err
	}

	return id, err
}
