package providers

import (
	"fmt"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
)

type Ingress struct {
	config config.Ingress
	client clients.ContainerTasks
	log    hclog.Logger
}

// NewIngress creates a new ingress provider
func NewIngress(c config.Ingress, cc clients.ContainerTasks, l hclog.Logger) *Ingress {
	return &Ingress{c, cc, l}
}

// Create the ingress
func (i *Ingress) Create() error {
	i.log.Info("Creating Ingress", "ref", i.config.Name)

	var serviceName string
	var volumes []config.Volume
	var env []config.KV
	command := make([]string, 0)

	switch v := i.config.TargetRef.(type) {
	case *config.Container:
		serviceName = utils.FQDN(v.Name, v.NetworkRef.Name)
	case *config.Cluster:

		// determine the type of cluster
		// if this is a k3s cluster we need to add the kubeconfig and
		// make sure that the proxy runs in kube mode
		if v.Driver == "k3s" {
			serviceName = i.config.Service
			_, _, kubeConfigPath := utils.CreateKubeConfigPath(v.Name)
			volumes = append(volumes, config.Volume{
				Source:      kubeConfigPath,
				Destination: "/.kube/kubeconfig.yml",
			})

			env = append(env, config.KV{Key: "KUBECONFIG", Value: "/.kube/kubeconfig.yml"})

			command = append(command, "--proxy-type")
			command = append(command, "kubernetes")

			// if the namespace is not present assume default
			if i.config.Namespace == "" {
				i.config.Namespace = "default"
			}

			command = append(command, "--namespace")
			command = append(command, i.config.Namespace)
		} else {
			serviceName = fmt.Sprintf("server.%s", utils.FQDN(v.Name, v.NetworkRef.Name))
		}

	default:
		return fmt.Errorf("Only Container ingress and K3s are supported at present")
	}

	image := "shipyardrun/ingress:latest"

	command = append(command, "--service-name")
	command = append(command, serviceName)

	// add the ports
	for _, p := range i.config.Ports {
		command = append(command, "--ports")
		command = append(command, fmt.Sprintf("%d:%d", p.Local, p.Remote))
	}

	// ingress simply crease a container with specific options
	c := config.Container{
		Name:        i.config.Name,
		NetworkRef:  i.config.NetworkRef,
		Ports:       i.config.Ports,
		Image:       config.Image{Name: image},
		Command:     command,
		Volumes:     volumes,
		Environment: env,
		IPAddress:   i.config.IPAddress,
	}

	_, err := i.client.CreateContainer(c)
	if err != nil {
		return err
	}

	// set the state
	i.config.State = config.Applied

	return nil
}

// Destroy the ingress
func (i *Ingress) Destroy() error {
	i.log.Info("Destroy Ingress", "ref", i.config.Name)

	ids, err := i.client.FindContainerIDs(i.config.Name, i.config.NetworkRef.Name)
	if err != nil {
		return err
	}

	for _, id := range ids {
		err := i.client.RemoveContainer(id)
		if err != nil {
			return err
		}

	}

	return nil
}

// Lookup the id of the ingress
func (i *Ingress) Lookup() ([]string, error) {
	return []string{}, nil
}

// Config returns the config for the provider
func (c *Ingress) Config() ConfigWrapper {
	return ConfigWrapper{"config.Ingress", c.config}
}

// State returns the state from the config
func (c *Ingress) State() config.State {
	return c.config.State
}

// SetState updates the state in the config
func (c *Ingress) SetState(state config.State) {
	c.config.State = state
}
