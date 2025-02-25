package ingress

import (
	"context"
	"fmt"
	"net"

	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/connector"
	"github.com/jumppad-labs/jumppad/pkg/clients/container"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/k8s"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/nomad"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	sdk "github.com/jumppad-labs/plugin-sdk"
)

var _ sdk.Provider = &Provider{}

// Ingress defines a provider for handling connection ingress for a cluster
type Provider struct {
	config    *Ingress
	client    container.ContainerTasks
	connector connector.Connector
	log       logger.Logger
}

func (p *Provider) Init(cfg htypes.Resource, l sdk.Logger) error {
	c, ok := cfg.(*Ingress)
	if !ok {
		return fmt.Errorf("unable to initialize Ingress provider, resource is not of type Ingress")
	}

	cli, err := clients.GenerateClients(l)
	if err != nil {
		return err
	}

	p.config = c
	p.client = cli.ContainerTasks
	p.connector = cli.Connector
	p.log = l

	return nil
}

func (p *Provider) Create(ctx context.Context) error {
	if ctx.Err() != nil {
		p.log.Debug("Skipping create, context cancelled", "ref", p.config.Meta.ID)
		return nil
	}

	p.log.Info("Create Ingress", "ref", p.config.Meta.ID)

	if p.config.ExposeLocal {
		return p.exposeLocal()
	}

	return p.exposeRemote()
}

// Destroy satisfies the interface method but is not implemented by LocalExec
func (p *Provider) Destroy(ctx context.Context, force bool) error {
	if ctx.Err() != nil {
		p.log.Debug("Skipping destroy, context cancelled", "ref", p.config.Meta.ID)
		return nil
	}

	p.log.Info("Destroy Ingress", "ref", p.config.Meta.ID, "id", p.config.IngressID)

	err := p.connector.RemoveService(p.config.IngressID)
	if err != nil {
		// fail silently as this should not stop us from destroying the
		// other resources
		p.log.Warn("Unable to remove local ingress", "ref", p.config.Meta.Name, "id", p.config.IngressID, "error", err)
	}

	return nil
}

// Lookup satisfies the interface method but is not implemented by LocalExec
func (p *Provider) Lookup() ([]string, error) {
	p.log.Debug("Lookup Ingress", "ref", p.config.Meta.ID, "id", p.config.IngressID)

	return []string{}, nil
}

func (p *Provider) Refresh(ctx context.Context) error {
	if ctx.Err() != nil {
		p.log.Debug("Skipping refresh, context cancelled", "ref", p.config.Meta.ID)
		return nil
	}

	p.log.Debug("Refresh Ingress", "ref", p.config.Meta.ID)

	return nil
}

func (p *Provider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.Meta.ID)

	return false, nil
}

func (p *Provider) exposeLocal() error {
	// validate the name
	if p.config.Target.Config["service"] == "connector" {
		return fmt.Errorf("unable to expose local service, Service name 'connector' is a reserved name")
	}

	// set the namespace
	p.config.Target.Config["namespace"] = "jumppad"

	remoteAddr := ""

	port := fmt.Sprintf("%d", p.config.Target.Port)

	if p.config.Target.NamedPort != "" {
		port = p.config.Target.NamedPort
	}

	switch p.config.Target.Resource.Meta.Type {
	case k8s.TypeK8sCluster:
		remoteAddr = fmt.Sprintf(
			"%s.%s.svc:%s",
			p.config.Target.Config["service"],
			p.config.Target.Config["namespace"],
			port,
		)
	case nomad.TypeNomadCluster:
		remoteAddr = fmt.Sprintf(
			"%s.%s.%s:%s",
			p.config.Target.Config["job"],
			p.config.Target.Config["group"],
			p.config.Target.Config["task"],
			port,
		)
	default:
		return fmt.Errorf("target type must be either a Kubernetes or a Nomad cluster")
	}

	// address of the remote connector
	connectorAddress := fmt.Sprintf("%s:%d", p.config.Target.Resource.ExternalIP, p.config.Target.Resource.ConnectorPort)

	// send the request
	p.log.Debug(
		"Calling connector to expose local service",
		"name", p.config.Target.Config["service"],
		"local_port", p.config.Target.Port,
		"connector_addr", connectorAddress,
		"remote_addr", fmt.Sprintf("localhost:%d", p.config.Port),
	)

	id, err := p.connector.ExposeService(
		p.config.Target.Config["service"],
		p.config.Target.Port,
		connectorAddress,
		fmt.Sprintf("localhost:%d", p.config.Port),
		"local",
	)

	if err != nil {
		return fmt.Errorf("unable to expose remote service on cluster :%w", err)
	}

	addr := fmt.Sprintf("%s:%d", utils.GetDockerIP(), p.config.Port)
	p.log.Debug("Successfully exposed service", "id", id, "dest", remoteAddr, "addr", addr)

	p.config.IngressID = id
	p.config.LocalAddress = addr
	p.config.RemoteAddress = remoteAddr

	return nil
}

func (p *Provider) exposeRemote() error {
	// check if the port is in use, if so, return an immediate error
	p.log.Debug("Checking if port is available", "port", p.config.Port)
	tc, err := net.Dial("tcp", fmt.Sprintf("0.0.0.0:%d", p.config.Port))
	if err == nil {
		p.log.Debug("Port in use", "port", p.config.Port)
		return fmt.Errorf("unable to create ingress port %d in use", p.config.Port)
	}

	if tc != nil {
		tc.Close()
	}

	destAddr := ""

	port := fmt.Sprintf("%d", p.config.Target.Port)

	if p.config.Target.NamedPort != "" {
		port = p.config.Target.NamedPort
	}

	switch p.config.Target.Resource.Meta.Type {
	case k8s.TypeK8sCluster:
		destAddr = fmt.Sprintf(
			"%s.%s.svc:%s",
			p.config.Target.Config["service"],
			p.config.Target.Config["namespace"],
			port,
		)
	case nomad.TypeNomadCluster:
		destAddr = fmt.Sprintf(
			"%s.%s.%s:%s",
			p.config.Target.Config["job"],
			p.config.Target.Config["group"],
			p.config.Target.Config["task"],
			port,
		)
	default:
		return fmt.Errorf("target type must be either a Kubernetes or a Nomad cluster")
	}

	// address of the remote connector
	connectorAddress := fmt.Sprintf("%s:%d", p.config.Target.Resource.ExternalIP, p.config.Target.Resource.ConnectorPort)

	// send the request
	p.log.Debug(
		"Calling connector to expose remote service",
		"name", p.config.Target.Config["service"],
		"local_port", p.config.Port,
		"connector_addr", connectorAddress,
		"remote_addr", destAddr,
	)

	id, err := p.connector.ExposeService(
		p.config.Target.Config["service"],
		p.config.Port,
		connectorAddress,
		destAddr,
		"remote",
	)

	if err != nil {
		return fmt.Errorf("unable to expose remote service on cluster :%w", err)
	}

	addr := fmt.Sprintf("%s:%d", utils.GetDockerIP(), p.config.Port)
	p.log.Debug("Successfully exposed service", "id", id, "dest", destAddr, "addr", addr)

	p.config.IngressID = id
	p.config.LocalAddress = addr
	p.config.RemoteAddress = destAddr

	return nil
}

// exposeK8sRemote exposes a remote kubernetes service to the local machine
//func (c *Ingress) exposeK8sRemote() error {
//	// get the target
//	res, err := c.config.ParentConfig.FindResource(c.config.Destination.Config.Cluster)
//	if err != nil {
//		return err
//	}
//
//	if c.config.Destination.Config.Address == "" {
//		return xerrors.Errorf("config parameter 'address' is required for destinations of type 'k8s'")
//	}
//
//	destAddr := fmt.Sprintf("%s:%s", c.config.Destination.Config.Address, c.config.Destination.Config.Port)
//
//	localPort, err := strconv.Atoi(c.config.Source.Config.Port)
//	if err != nil {
//		return xerrors.Errorf("Unable to parse remote port :%w", err)
//	}
//
//	if localPort == 30001 || localPort == 30002 {
//		return fmt.Errorf("unable to expose local service using remote port %d,"+
//			"ports 30001 and 30002 are reserved for internal use", localPort)
//	}
//
//	// sanitize the name to make it uri format
//	serviceName, err := utils.ReplaceNonURIChars(c.config.Name)
//	if err != nil {
//		return xerrors.Errorf("unable to replace non URI characters in service name %s :%w", c.config.Name, err)
//	}
//
//	connectorAddress := fmt.Sprintf("%s:%d", res.(*resources.K8sCluster).ExternalIP, res.(*resources.K8sCluster).ConnectorPort)
//
//	// send the request
//	c.log.Debug(
//		"Calling connector to expose remote service",
//		"name", serviceName,
//		"local_port", localPort,
//		"connector_addr", connectorAddress,
//		"local_addr", destAddr,
//	)
//
//	id, err := c.connector.ExposeService(
//		serviceName,
//		localPort,
//		connectorAddress,
//		destAddr,
//		"remote")
//
//	if err != nil {
//		return xerrors.Errorf("unable to expose remote cluster service to local machine :%w", err)
//	}
//
//	local, _ := utils.GetLocalIPAndHostname()
//	addr := fmt.Sprintf("%s:%d", local, localPort)
//
//	c.log.Debug("Successfully exposed service", "id", id, "addr", addr)
//
//	c.config.IngressID = id
//	c.config.Address = addr
//
//	return nil
//}
