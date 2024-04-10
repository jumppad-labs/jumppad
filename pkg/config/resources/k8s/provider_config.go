package k8s

import (
	"context"
	"fmt"
	"os"
	"time"

	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/k8s"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	sdk "github.com/jumppad-labs/plugin-sdk"
	"golang.org/x/xerrors"
)

var _ sdk.Provider = &ConfigProvider{}

type ConfigProvider struct {
	config *K8sConfig
	client k8s.Kubernetes
	log    sdk.Logger
}

func (p *ConfigProvider) Init(cfg htypes.Resource, l sdk.Logger) error {
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
func (p *ConfigProvider) Create(ctx context.Context) error {
	if ctx.Err() != nil {
		p.log.Debug("Skipping create, context cancelled", "ref", p.config.Meta.ID)
		return nil
	}

	p.log.Info("Applying Kubernetes configuration", "ref", p.config.Meta.Name, "config", p.config.Paths)

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

		err = p.client.HealthCheckPods(ctx, p.config.HealthCheck.Pods, to)
		if err != nil {
			return xerrors.Errorf("healthcheck failed after helm chart setup: %w", err)
		}
	}

	// set the checksums
	cs, err := p.generateChecksums()
	if err != nil {
		return xerrors.Errorf("unable to generate checksums: %w", err)
	}

	p.config.JobChecksums = cs

	return nil
}

// Destroy the Kubernetes resources defined by the config
func (p *ConfigProvider) Destroy(ctx context.Context, force bool) error {
	if ctx.Err() != nil {
		p.log.Debug("Skipping destroy, context cancelled", "ref", p.config.Meta.ID)
		return nil
	}

	p.log.Info("Destroy Kubernetes configuration", "ref", p.config.Meta.ID, "config", p.config.Paths)

	err := p.setup()
	if err != nil {
		return err
	}

	err = p.client.Delete(p.config.Paths)
	if err != nil {
		p.log.Debug("There was a problem destroying Kubernetes config, logging message but ignoring error", "ref", p.config.Meta.ID, "error", err)
	}
	return nil
}

// Lookup the Kubernetes resources defined by the config
func (p *ConfigProvider) Lookup() ([]string, error) {
	return []string{}, nil
}

func (p *ConfigProvider) Refresh(ctx context.Context) error {
	cp, err := p.getChangedPaths()
	if err != nil {
		return err
	}

	if len(cp) < 1 {
		return nil
	}

	p.log.Info("Refresh Kubernetes config", "ref", p.config.Meta.ID, "paths", cp)

	err = p.Destroy(ctx, false)
	if err != nil {
		return err
	}

	return p.Create(ctx)
}

func (p *ConfigProvider) Changed() (bool, error) {
	cp, err := p.getChangedPaths()
	if err != nil {
		return false, err
	}

	if len(cp) > 0 {
		p.log.Debug("Kubernetes jobs changed, needs refresh", "ref", p.config.Meta.ID)
		return true, nil
	}
	return false, nil
}

func (p *ConfigProvider) setup() error {
	var err error
	p.client, err = p.client.SetConfig(p.config.Cluster.KubeConfig.ConfigPath)
	if err != nil {
		return xerrors.Errorf("unable to create Kubernetes client: %w", err)
	}

	return nil
}

// generateChecksums generates a sha256 checksum for each of the the paths
func (p *ConfigProvider) generateChecksums() ([]string, error) {
	checksums := []string{}

	for _, p := range p.config.Paths {
		f, err := os.Open(p)
		if err != nil {
			return nil, err
		}

		fi, err := f.Stat()
		if err != nil {
			return nil, err
		}

		var hash string

		if fi.IsDir() {
			hash, err = utils.HashDir(p)
		} else {
			hash, err = utils.HashFile(p)
		}

		if err != nil {
			return nil, err
		}

		checksums = append(checksums, hash)
	}

	return checksums, nil
}

// getChangedPaths returns the paths that have changed since the nomad jobs
// were last applied
func (p *ConfigProvider) getChangedPaths() ([]string, error) {
	// get the checksums
	cs, err := p.generateChecksums()
	if err != nil {
		return nil, err
	}

	// if we have more checksums than previous assume everything has changed
	if len(p.config.JobChecksums) != len(cs) {
		return p.config.Paths, nil
	}

	// compare the checksums
	diff := []string{}
	for i, c := range p.config.JobChecksums {

		if c != cs[i] {
			diff = append(diff, p.config.Paths[i])
		}
	}

	return diff, nil
}
