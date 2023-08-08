package k8s

import (
	"fmt"
	"time"

	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"golang.org/x/xerrors"
)

type ConfigProvider struct {
	config *K8sConfig
	client clients.Kubernetes
	log    logger.Logger
}

func (p *ConfigProvider) Init(cfg htypes.Resource, l logger.Logger) error {
	c, ok := cfg.(*K8sConfig)
	if !ok {
		return fmt.Errorf("unable to initialize Config provider, resource is not of type K8sConfig")
	}

	cli, err := clients.GenerateClients(l)
	if err != nil {
		return err
	}

	p.config = c
	p.client = cli.Kubernetes
	p.log = l

	return nil
}

// Create the Kubernetes resources defined by the config
func (p *ConfigProvider) Create() error {
	p.log.Info("Applying Kubernetes configuration", "ref", p.config.Name, "config", p.config.Paths)

	err := p.setup()
	if err != nil {
		return err
	}

	err = p.client.Apply(p.config.Paths, p.config.WaitUntilReady)
	if err != nil {
		return err
	}

	// run any health checks
	if p.config.HealthCheck != nil && len(p.config.HealthCheck.Pods) > 0 {
		to, err := time.ParseDuration(p.config.HealthCheck.Timeout)
		if err != nil {
			return xerrors.Errorf("unable to parse healthcheck duration: %w", err)
		}

		err = p.client.HealthCheckPods(p.config.HealthCheck.Pods, to)
		if err != nil {
			return xerrors.Errorf("healthcheck failed after helm chart setup: %w", err)
		}
	}

	return nil
}

// Destroy the Kubernetes resources defined by the config
func (p *ConfigProvider) Destroy() error {
	p.log.Info("Destroy Kubernetes configuration", "ref", p.config.ID, "config", p.config.Paths)

	err := p.setup()
	if err != nil {
		return err
	}

	err = p.client.Delete(p.config.Paths)
	if err != nil {
		p.log.Debug("There was a problem destroying Kubernetes config, logging message but ignoring error", "ref", p.config.ID, "error", err)
	}
	return nil
}

// Lookup the Kubernetes resources defined by the config
func (p *ConfigProvider) Lookup() ([]string, error) {
	return []string{}, nil
}

func (p *ConfigProvider) Refresh() error {
	p.log.Debug("Refresh Kubernetes configuration", "ref", p.config.ID)

	return nil
}

func (p *ConfigProvider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.ID)

	return false, nil
}

func (p *ConfigProvider) setup() error {
	var err error
	p.client, err = p.client.SetConfig(p.config.Cluster.KubeConfig)
	if err != nil {
		return xerrors.Errorf("unable to create Kubernetes client: %w", err)
	}

	return nil
}
