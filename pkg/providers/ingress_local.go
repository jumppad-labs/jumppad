package providers

import (
	"fmt"
	"strconv"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"golang.org/x/xerrors"
)

// Ingress defines a provider for handling connection ingress for a cluster
type IngressLocal struct {
	config    *config.LocalIngress
	client    clients.ContainerTasks
	connector clients.Connector
	log       hclog.Logger
}

// NewIngress creates a new ingress provider
func NewIngressLocal(
	c *config.LocalIngress,
	cc clients.ContainerTasks,
	co clients.Connector,
	l hclog.Logger) *IngressLocal {

	return &IngressLocal{c, cc, co, l}
}

func (c *IngressLocal) Create() error {
	c.log.Info("Create Local Ingress", "ref", c.config.Name)

	// get the target
	res, err := c.config.FindDependentResource(c.config.Target)
	if err != nil {
		return err
	}

	// validate the name
	if c.config.Name == "connector" {
		return fmt.Errorf("Service name 'connector' is a reserved name")
	}

	// validate the remote port, can not be 60000 or 60001 as these
	// ports are used by the connector service
	remotePort, err := strconv.Atoi(c.config.Ports[0].Remote)
	if err != nil {
		return xerrors.Errorf("Unable to parse remote port :%w", err)
	}

	if remotePort == 60000 || remotePort == 60001 {
		return fmt.Errorf("Unable to expose local service using remote port %d,"+
			"ports 60000 and 60001 are reserved for internal use", remotePort)
	}

	// get the address of the remote connector from the target
	_, configPath := utils.CreateClusterConfigPath(res.Info().Name)

	cc := &clients.ClusterConfig{}
	err = cc.Load(configPath, clients.LocalContext)
	if err != nil {
		return xerrors.Errorf("Unable to load cluster config :%w", err)
	}

	// send the request
	c.log.Debug(
		"Calling connector to expose local service",
		"name", c.config.Name,
		"remote_port", remotePort,
		"connector_addr", cc.ConnectorAddress(),
		"local_addr", fmt.Sprintf("%s:%s", c.config.Destination, c.config.Ports[0].Local),
	)

	id, err := c.connector.ExposeService(
		c.config.Name,
		remotePort,
		cc.ConnectorAddress(),
		fmt.Sprintf("%s:%s", c.config.Destination, c.config.Ports[0].Local),
		"local")

	if err != nil {
		return xerrors.Errorf("Unable to expose local service to remote cluster :%w", err)
	}

	c.log.Debug("Successfully exposed service", "id", id)
	c.config.Id = id

	return nil
}

// Destroy statisfies the interface method but is not implemented by LocalExec
func (c *IngressLocal) Destroy() error {
	c.log.Info("Destroy Local Ingress", "ref", c.config.Name, "id", c.config.Id)

	err := c.connector.RemoveService(c.config.Id)
	if err != nil {
		// fail silently as this should not stop us from destroying the
		// other resources
		c.log.Warn("Unable to remove local ingress", "error", err)
	}

	return nil
}

// Lookup statisfies the interface method but is not implemented by LocalExec
func (c *IngressLocal) Lookup() ([]string, error) {
	return []string{}, nil
}
