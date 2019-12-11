package providers

import (
	"fmt"

	"github.com/shipyard-run/cli/pkg/clients"
	"github.com/shipyard-run/cli/pkg/config"
)

type Ingress struct {
	config *config.Ingress
	client clients.Docker
}

func NewIngress(c *config.Ingress, cc clients.Docker) *Ingress {
	return &Ingress{c, cc}
}

func (i *Ingress) Create() error {
	// get the target ref
	t, ok := i.config.TargetRef.(*config.Container)
	if !ok {
		return fmt.Errorf("Only Container ingress is supported at present")
	}

	image := "shipyardrun/ingress:latest"
	command := make([]string, 0)

	// --network onprem docker.pkg.github.com/shipyard-run/ingress:latest --service-name consul.onprem.shipyard --port-remote 8500 --port-host 8500t
	// build the command based on the ports
	command = append(command, "--service-name")
	command = append(command, FQDN(t.Name, t.NetworkRef.Name))

	// add the ports
	for _, p := range i.config.Ports {
		command = append(command, "--ports")
		command = append(command, fmt.Sprintf("%d:%d", p.Local, p.Remote))
	}

	// ingress simply crease a container with specific options
	c := &config.Container{
		Name:       i.config.Name,
		NetworkRef: i.config.NetworkRef,
		Ports:      i.config.Ports,
		Image:      image,
		Command:    command,
	}

	p := NewContainer(c, i.client)

	return p.Create()
}

func (i *Ingress) Destroy() error {
	c := &config.Container{
		Name:       i.config.Name,
		NetworkRef: i.config.NetworkRef,
	}

	p := NewContainer(c, i.client)

	return p.Destroy()
}

func (i *Ingress) Lookup() (string, error) {
	return "", nil
}
