package providers

import (
	"fmt"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
)

type Ingress struct {
	config *config.Ingress
	client clients.Docker
	log    hclog.Logger
}

func NewIngress(c *config.Ingress, cc clients.Docker, l hclog.Logger) *Ingress {
	return &Ingress{c, cc, l}
}

func (i *Ingress) Create() error {
	i.log.Info("Creating Ingress", "ref", i.config.Name)

	var serviceName string
	var volumes []config.Volume
	var env []config.KV
	command := make([]string, 0)

	switch v := i.config.TargetRef.(type) {
	case *config.Container:
		serviceName = FQDN(v.Name, v.NetworkRef)
	case *config.Cluster:
		serviceName = i.config.Service

		_, _, kubeConfigPath := CreateKubeConfigPath(v.Name)
		volumes = append(volumes, config.Volume{
			Source:      kubeConfigPath,
			Destination: "/.kube/kubeconfig.yml",
		})

		env = append(env, config.KV{Key: "KUBECONFIG", Value: "/.kube/kubeconfig.yml"})

		command = append(command, "--proxy-type")
		command = append(command, "kubernetes")
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
	c := &config.Container{
		Name:        i.config.Name,
		NetworkRef:  i.config.NetworkRef,
		Ports:       i.config.Ports,
		Image:       config.Image{Name: image},
		Command:     command,
		Volumes:     volumes,
		Environment: env,
	}

	p := NewContainer(c, i.client, i.log.With("parent_ref", i.config.Name))

	return p.Create()
}

// Destroy the ingress
func (i *Ingress) Destroy() error {
	i.log.Info("Destroy Ingress", "ref", i.config.Name)

	c := &config.Container{
		Name:       i.config.Name,
		NetworkRef: i.config.NetworkRef,
	}

	p := NewContainer(c, i.client, i.log.With("parent_ref", i.config.Name))

	return p.Destroy()
}

// Lookup the id of the ingress
func (i *Ingress) Lookup() (string, error) {
	return "", nil
}
