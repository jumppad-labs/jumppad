package helm

import (
	"context"
	"fmt"
	"time"

	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/getter"
	"github.com/jumppad-labs/jumppad/pkg/clients/helm"
	"github.com/jumppad-labs/jumppad/pkg/clients/k8s"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	sdk "github.com/jumppad-labs/plugin-sdk"
	"golang.org/x/xerrors"
)

var _ sdk.Provider = &Provider{}

type Provider struct {
	config       *Helm
	kubeClient   k8s.Kubernetes
	helmClient   helm.Helm
	getterClient getter.Getter
	log          logger.Logger
}

func (p *Provider) Init(cfg htypes.Resource, l sdk.Logger) error {
	h, ok := cfg.(*Helm)

	if !ok {
		return fmt.Errorf("unable to initialize Helm provider, resource is not of type Helm")
	}

	p.config = h

	cli, err := clients.GenerateClients(l)
	if err != nil {
		return err
	}

	p.config = h
	p.kubeClient = cli.Kubernetes
	p.helmClient = cli.Helm
	p.getterClient = cli.Getter
	p.log = l

	return nil
}

// Create implements the provider Create method
func (p *Provider) Create(ctx context.Context) error {
	if ctx.Err() != nil {
		p.log.Debug("Skipping create, context cancelled", "ref", p.config.Meta.ID)
		return nil
	}

	p.log.Info("Creating Helm chart", "ref", p.config.Meta.ID)

	// if the namespace is null set to default
	if p.config.Namespace == "" {
		p.config.Namespace = "default"
	}

	// is this chart ot be loaded from a repository?
	if p.config.Repository != nil {
		p.log.Debug("Updating Helm chart repository", "name", p.config.Repository.Name, "url", p.config.Repository.URL)

		err := p.helmClient.UpsertChartRepository(p.config.Repository.Name, p.config.Repository.URL)
		if err != nil {
			return xerrors.Errorf("unable to initialize chart repository: %w", err)
		}
	}

	// is the source a helm repo which should be downloaded?
	if !utils.IsLocalFolder(p.config.Chart) && p.config.Repository == nil {
		p.log.Debug("Fetching remote Helm chart", "ref", p.config.Meta.Name, "chart", p.config.Chart)

		helmFolder := utils.HelmLocalFolder(p.config.Chart)

		err := p.getterClient.Get(p.config.Chart, helmFolder)
		if err != nil {
			return xerrors.Errorf("Unable to download remote chart: %w", err)
		}

		// set the config to the local path
		p.config.Chart = helmFolder
	}

	// set the KubeConfig for the kubernetes client
	// this is used by the health checks
	var err error
	p.log.Debug("Using Kubernetes config", "ref", p.config.Meta.ID, "path", p.config.Cluster.KubeConfig)
	p.kubeClient, err = p.kubeClient.SetConfig(p.config.Cluster.KubeConfig.ConfigPath)
	if err != nil {
		return xerrors.Errorf("unable to create Kubernetes client: %w", err)
	}

	// sanitize the chart name
	newName, _ := utils.ReplaceNonURIChars(p.config.Meta.Name)

	failCount := 0

	to := time.Duration(300 * time.Second)
	if p.config.Timeout != "" {
		to, err = time.ParseDuration(p.config.Timeout)
		if err != nil {
			return xerrors.Errorf("unable to parse timeout duration: %w", err)
		}
	}

	timeout := time.After(to)
	errChan := make(chan error)
	doneChan := make(chan struct{})

	go func() {
		for {

			// context is cancelled do not retry
			if ctx.Err() != nil {
				p.log.Debug("Skipping create, context cancelled", "ref", p.config.Meta.ID)
				err := fmt.Errorf("context cancelled, skipping helm chart creation, ref: %s", p.config.Meta.ID)
				errChan <- err
			}

			err = p.helmClient.Create(
				p.config.Cluster.KubeConfig.ConfigPath,
				newName,
				p.config.Namespace,
				p.config.CreateNamespace,
				p.config.SkipCRDs,
				p.config.Chart,
				p.config.Version,
				p.config.Values,
				p.config.ValuesString)

			if err == nil {
				doneChan <- struct{}{}
				break
			}

			failCount++

			if failCount >= p.config.Retry {
				errChan <- err
			} else {
				p.log.Debug("Chart apply failed, retrying", "error", err)
				time.Sleep(5 * time.Second)
			}
		}
	}()

	select {
	case <-timeout:
		return xerrors.Errorf("timeout waiting for helm chart to complete")
	case createErr := <-errChan:
		return createErr
	case <-doneChan:
		p.log.Debug("Helm chart applied", "ref", p.config.Meta.Name)
	}

	// we can now health check the install
	if p.config.HealthCheck != nil && len(p.config.HealthCheck.Pods) > 0 {
		to, err := time.ParseDuration(p.config.HealthCheck.Timeout)
		if err != nil {
			return xerrors.Errorf("unable to parse health check duration: %w", err)
		}

		err = p.kubeClient.HealthCheckPods(ctx, p.config.HealthCheck.Pods, to)
		if err != nil {
			return xerrors.Errorf("health check failed after helm chart setup: %w", err)
		}
	}

	return nil
}

// Destroy implements the provider Destroy method
func (p *Provider) Destroy(ctx context.Context, force bool) error {
	if ctx.Err() != nil {
		p.log.Debug("Skipping destroy, context cancelled", "ref", p.config.Meta.ID)
		return nil
	}

	p.log.Info("Destroy Helm chart", "ref", p.config.Meta.ID)

	// if the namespace is null set to default
	if p.config.Namespace == "" {
		p.config.Namespace = "default"
	}

	// sanitize the chart name
	newName, _ := utils.ReplaceNonURIChars(p.config.Meta.Name)

	// get the target cluster
	err := p.helmClient.Destroy(p.config.Cluster.KubeConfig.ConfigPath, newName, p.config.Namespace)

	if err != nil {
		p.log.Warn("There was a problem destroying Helm chart, logging message but ignoring error", "ref", p.config.Meta.ID, "error", err)
	}

	return nil
}

// Lookup implements the provider Lookup method
func (p *Provider) Lookup() ([]string, error) {
	return []string{}, nil
}

func (p *Provider) Refresh(ctx context.Context) error {
	if ctx.Err() != nil {
		p.log.Debug("Skipping refresh, context cancelled", "ref", p.config.Meta.ID)
		return nil
	}

	p.log.Debug("Refresh Helm Chart", "ref", p.config.Meta.Name)

	return nil
}

func (p *Provider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.Meta.Name)

	return false, nil
}
