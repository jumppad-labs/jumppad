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
type Ingress struct {
	config    *config.Ingress
	client    clients.ContainerTasks
	connector clients.Connector
	log       hclog.Logger
}

// NewIngress creates a new ingress provider
func NewIngress(
	c *config.Ingress,
	cc clients.ContainerTasks,
	co clients.Connector,
	l hclog.Logger) *Ingress {

	return &Ingress{c, cc, co, l}
}

func (c *Ingress) Create() error {
	c.log.Info("Create Ingress", "ref", c.config.Name)

	if c.config.Destination.Driver == "local" {
		return c.exposeLocal()
	}

	if c.config.Destination.Driver == "k8s" {
		return c.exposeK8sRemote()
	}

	return nil
}

// Destroy statisfies the interface method but is not implemented by LocalExec
func (c *Ingress) Destroy() error {
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
func (c *Ingress) Lookup() ([]string, error) {

	return []string{}, nil
}

func (c *Ingress) exposeLocal() error {
	// get the target
	res, err := c.config.FindDependentResource(c.config.Source.Config.Cluster)
	if err != nil {
		return err
	}

	// validate the name
	if c.config.Source.Config.Service == "connector" {
		return fmt.Errorf("Service name 'connector' is a reserved name")
	}

	// validate the remote port, can not be 60000 or 60001 as these
	// ports are used by the connector service
	remotePort, err := strconv.Atoi(c.config.Source.Config.Port)
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

	destAddr := fmt.Sprintf("%s:%s", c.config.Destination.Config.Service, c.config.Destination.Config.Port)

	// send the request
	c.log.Debug(
		"Calling connector to expose local service",
		"name", c.config.Source.Config.Service,
		"remote_port", remotePort,
		"connector_addr", cc.ConnectorAddress(),
		"local_addr", destAddr,
	)

	id, err := c.connector.ExposeService(
		c.config.Source.Config.Service,
		remotePort,
		cc.ConnectorAddress(),
		destAddr,
		"local")

	if err != nil {
		return xerrors.Errorf("Unable to expose local service to remote cluster :%w", err)
	}

	c.log.Debug("Successfully exposed service", "id", id)
	c.config.Id = id

	return nil
}

func (c *Ingress) exposeK8sRemote() error {
	// get the target
	res, err := c.config.FindDependentResource(c.config.Destination.Config.Cluster)
	if err != nil {
		return err
	}

	// get the address of the remote connector from the target
	_, configPath := utils.CreateClusterConfigPath(res.Info().Name)

	cc := &clients.ClusterConfig{}
	err = cc.Load(configPath, clients.LocalContext)
	if err != nil {
		return xerrors.Errorf("Unable to load cluster config :%w", err)
	}

	namespace := "default"
	if c.config.Destination.Config.Namespace != "" {
		namespace = c.config.Destination.Config.Namespace
	}

	destAddr := fmt.Sprintf("%s.%s.svc.cluster.local:%s", c.config.Destination.Config.Service, namespace, c.config.Destination.Config.Port)

	localPort, err := strconv.Atoi(c.config.Source.Config.Port)
	if err != nil {
		return xerrors.Errorf("Unable to parse remote port :%w", err)
	}

	if localPort == 30001 || localPort == 30002 {
		return fmt.Errorf("Unable to expose local service using remote port %d,"+
			"ports 30001 and 30002 are reserved for internal use", localPort)
	}

	// send the request
	c.log.Debug(
		"Calling connector to expose remote service",
		"name", c.config.Destination.Config.Service,
		"local_port", localPort,
		"connector_addr", cc.ConnectorAddress(),
		"local_addr", destAddr,
	)

	id, err := c.connector.ExposeService(
		c.config.Destination.Config.Service,
		localPort,
		cc.ConnectorAddress(),
		destAddr,
		"remote")

	if err != nil {
		return xerrors.Errorf("Unable to expose local service to remote cluster :%w", err)
	}

	c.log.Debug("Successfully exposed service", "id", id)
	c.config.Id = id

	return nil
}
