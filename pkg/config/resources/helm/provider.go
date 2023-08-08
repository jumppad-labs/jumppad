package helm

import (
	"fmt"
	"time"

	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"golang.org/x/xerrors"
)

type Provider struct {
	config       *Helm
	kubeClient   clients.Kubernetes
	helmClient   clients.Helm
	getterClient clients.Getter
	log          logger.Logger
}

func (p *Provider) Init(cfg htypes.Resource, l logger.Logger) error {
	h, ok := cfg.(*Helm)

	if !ok {
		return fmt.Errorf("unable to initialize Helm provider, resource is not of type Helm")
	}

	p.config = h

	cli, err := clients.GenerateClients(l)
	if err != nil {
		return err
	}

	p.config = c
	p.kubeClient = cli.Kubernetes
	p.helmClient = cli.Helm
	p.getterClient = cli.Getter
	p.log = l

	return nil
}

// Create implements the provider Create method
func (p *Provider) Create() error {
	p.log.Info("Creating Helm chart", "ref", p.config.ID)

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
		p.log.Debug("Fetching remote Helm chart", "ref", p.config.Name, "chart", p.config.Chart)

		helmFolder := utils.GetHelmLocalFolder(p.config.Chart)

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
	p.log.Debug("Using Kubernetes config", "ref", p.config.ID, "path", p.config.K8sConfig)
	p.kubeClient, err = p.kubeClient.SetConfig(p.config.K8sConfig)
	if err != nil {
		return xerrors.Errorf("unable to create Kubernetes client: %w", err)
	}

	// sanitize the chart name
	newName, _ := utils.ReplaceNonURIChars(p.config.Name)

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
			err = p.helmClient.Create(
				p.config.K8sConfig,
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
		p.log.Debug("Helm chart applied", "ref", p.config.Name)
	}

	// we can now health check the install
	if p.config.HealthCheck != nil && len(p.config.HealthCheck.Pods) > 0 {
		to, err := time.ParseDuration(p.config.HealthCheck.Timeout)
		if err != nil {
			return xerrors.Errorf("unable to parse health check duration: %w", err)
		}

		err = p.kubeClient.HealthCheckPods(p.config.HealthCheck.Pods, to)
		if err != nil {
			return xerrors.Errorf("health check failed after helm chart setup: %w", err)
		}
	}

	return nil
}

// Destroy implements the provider Destroy method
func (p *Provider) Destroy() error {
	p.log.Info("Destroy Helm chart", "ref", p.config.ID)

	// if the namespace is null set to default
	if p.config.Namespace == "" {
		p.config.Namespace = "default"
	}

	// sanitize the chart name
	newName, _ := utils.ReplaceNonURIChars(p.config.Name)

	// get the target cluster
	err := p.helmClient.Destroy(p.config.K8sConfig, newName, p.config.Namespace)

	if err != nil {
		p.log.Debug("There was a problem destroying Helm chart, logging message but ignoring error", "ref", p.config.ID, "error", err)
	}

	return nil
}

// Lookup implements the provider Lookup method
func (p *Provider) Lookup() ([]string, error) {
	return []string{}, nil
}

func (p *Provider) Refresh() error {
	p.log.Debug("Refresh Helm Chart", "ref", p.config.Name)

	return nil
}

func (p *Provider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.Name)

	return false, nil
}
