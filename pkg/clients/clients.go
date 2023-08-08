package clients

import (
	"time"

	"github.com/jumppad-labs/jumppad/pkg/clients/command"
	"github.com/jumppad-labs/jumppad/pkg/clients/connector"
	"github.com/jumppad-labs/jumppad/pkg/clients/container"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/clients/system"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

type Clients struct {
	Docker         container.Docker
	ContainerTasks container.ContainerTasks
	Kubernetes     Kubernetes
	Helm           Helm
	HTTP           HTTP
	Nomad          Nomad
	Command        command.Command
	Logger         logger.Logger
	Getter         Getter
	Browser        system.System
	ImageLog       ImageLog
	Connector      connector.Connector
	TarGz          *TarGz
}

// GenerateClients creates the various clients for creating and destroying resources
func GenerateClients(l logger.Logger) (*Clients, error) {
	dc, err := container.NewDocker()
	if err != nil {
		return nil, err
	}

	kc := NewKubernetes(60*time.Second, l)

	hec := NewHelm(l)

	ec := command.NewCommand(30*time.Second, l)

	hc := NewHTTP(1*time.Second, l)

	nc := NewNomad(hc, 1*time.Second, l)

	bp := NewGetter(false)

	bc := &system.SystemImpl{}

	il := NewImageFileLog(utils.ImageCacheLog())

	tgz := &TarGz{}

	ct := container.NewDockerTasks(dc, il, tgz, l)

	co := connector.DefaultConnectorOptions()
	cc := connector.NewConnector(co)

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
