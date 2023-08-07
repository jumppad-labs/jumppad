package jumppad

import (
	"time"

	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/container"
	"github.com/jumppad-labs/jumppad/pkg/clients/system"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

type Clients struct {
	Docker         container.Docker
	ContainerTasks container.ContainerTasks
	Kubernetes     clients.Kubernetes
	Helm           clients.Helm
	HTTP           clients.HTTP
	Nomad          clients.Nomad
	Command        clients.Command
	Logger         clients.Logger
	Getter         clients.Getter
	Browser        system.System
	ImageLog       clients.ImageLog
	Connector      clients.Connector
	TarGz          *clients.TarGz
}

// GenerateClients creates the various clients for creating and destroying resources
func GenerateClients(l clients.Logger) (*Clients, error) {
	dc, err := container.NewDocker()
	if err != nil {
		return nil, err
	}

	kc := clients.NewKubernetes(60*time.Second, l)

	hec := clients.NewHelm(l)

	ec := clients.NewCommand(30*time.Second, l)

	hc := clients.NewHTTP(1*time.Second, l)

	nc := clients.NewNomad(hc, 1*time.Second, l)

	bp := clients.NewGetter(false)

	bc := &system.SystemImpl{}

	il := clients.NewImageFileLog(utils.ImageCacheLog())

	tgz := &clients.TarGz{}

	ct := container.NewDockerTasks(dc, il, tgz, l)

	co := clients.DefaultConnectorOptions()
	cc := clients.NewConnector(co)

	return &Clients{
		ContainerTasks: ct,
		Docker:         dc,
		Kubernetes:     kc,
		Helm:           hec,
		Command:        ec,
		HTTP:           hc,
		Nomad:          nc,
		Logger:         l,
		Getter:         bp,
		Browser:        bc,
		ImageLog:       il,
		Connector:      cc,
		TarGz:          tgz,
	}, nil
}
