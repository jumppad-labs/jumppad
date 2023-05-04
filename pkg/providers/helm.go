package providers

import (
	"time"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"golang.org/x/xerrors"
)

type Helm struct {
	config       *resources.Helm
	kubeClient   clients.Kubernetes
	helmClient   clients.Helm
	getterClient clients.Getter
	log          hclog.Logger
}

// NewHelm creates a new Helm provider
func NewHelm(c *resources.Helm, kc clients.Kubernetes, hc clients.Helm, g clients.Getter, l hclog.Logger) *Helm {
	return &Helm{c, kc, hc, g, l}
}

// Create implements the provider Create method
func (h *Helm) Create() error {
	h.log.Info("Creating Helm chart", "ref", h.config.ID)

	// get the target cluster
	kcPath, err := h.getKubeConfigPath()
	if err != nil {
		return err
	}

	// if the namespace is null set to default
	if h.config.Namespace == "" {
		h.config.Namespace = "default"
	}

	// is this chart ot be loaded from a repository?
	if h.config.Repository != nil {
		h.log.Debug("Updating Helm chart repository", "name", h.config.Repository.Name, "url", h.config.Repository.URL)

		err := h.helmClient.UpsertChartRepository(h.config.Repository.Name, h.config.Repository.URL)
		if err != nil {
			return xerrors.Errorf("unable to initialize chart repository: %w", err)
		}
	}

	// is the source a helm repo which should be downloaded?
	if !utils.IsLocalFolder(h.config.Chart) && h.config.Repository == nil {
		h.log.Debug("Fetching remote Helm chart", "ref", h.config.Name, "chart", h.config.Chart)

		helmFolder := utils.GetHelmLocalFolder(h.config.Chart)

		err := h.getterClient.Get(h.config.Chart, helmFolder)
		if err != nil {
			return xerrors.Errorf("Unable to download remote chart: %w", err)
		}

		// set the config to the local path
		h.config.Chart = helmFolder
	}

	// set the KubeConfig for the kubernetes client
	// this is used by the health checks
	h.log.Debug("Using Kubernetes config", "ref", h.config.ID, "path", kcPath)
	h.kubeClient, err = h.kubeClient.SetConfig(kcPath)
	if err != nil {
		return xerrors.Errorf("unable to create Kubernetes client: %w", err)
	}

	// sanitize the chart name
	newName, _ := utils.ReplaceNonURIChars(h.config.Name)

	failCount := 0

	to := time.Duration(300 * time.Second)
	if h.config.Timeout != "" {
		to, err = time.ParseDuration(h.config.Timeout)
		if err != nil {
			return xerrors.Errorf("unable to parse timeout duration: %w", err)
		}
	}

	timeout := time.After(to)
	errChan := make(chan error)
	doneChan := make(chan struct{})

	go func() {
		for {
			err = h.helmClient.Create(
				kcPath,
				newName,
				h.config.Namespace,
				h.config.CreateNamespace,
				h.config.SkipCRDs,
				h.config.Chart,
				h.config.Version,
				h.config.Values,
				h.config.ValuesString)

			if err == nil {
				doneChan <- struct{}{}
				break
			}

			failCount++

			if failCount >= h.config.Retry {
				errChan <- err
			} else {
				h.log.Debug("Chart apply failed, retrying", "error", err)
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
		h.log.Debug("Helm chart applied", "ref", h.config.Name)
	}

	// we can now health check the install
	if h.config.HealthCheck != nil && len(h.config.HealthCheck.Pods) > 0 {
		to, err := time.ParseDuration(h.config.HealthCheck.Timeout)
		if err != nil {
			return xerrors.Errorf("unable to parse health check duration: %w", err)
		}

		err = h.kubeClient.HealthCheckPods(h.config.HealthCheck.Pods, to)
		if err != nil {
			return xerrors.Errorf("health check failed after helm chart setup: %w", err)
		}
	}

	return nil
}

// Destroy implements the provider Destroy method
func (h *Helm) Destroy() error {
	h.log.Info("Destroy Helm chart", "ref", h.config.Name)
	kcPath, err := h.getKubeConfigPath()
	if err != nil {
		return err
	}

	// if the namespace is null set to default
	if h.config.Namespace == "" {
		h.config.Namespace = "default"
	}

	// sanitize the chart name
	newName, _ := utils.ReplaceNonURIChars(h.config.Name)

	// get the target cluster
	h.helmClient.Destroy(kcPath, newName, h.config.Namespace)

	if err != nil {
		h.log.Debug("There was a problem destroying Helm chart, logging message but ignoring error", "ref", h.config.Name, "error", err)
	}

	return nil
}

// Lookup implements the provider Lookup method
func (h *Helm) Lookup() ([]string, error) {
	return []string{}, nil
}

func (c *Helm) Refresh() error {
	c.log.Info("Refresh Helm Chart", "ref", c.config.Name)

	return nil
}

func (h *Helm) getKubeConfigPath() (string, error) {
	target, err := h.config.ParentConfig.FindResource(h.config.Cluster)
	if err != nil {
		return "", xerrors.Errorf("Unable to find cluster: %w", err)
	}

	return target.(*resources.K8sCluster).KubeConfig, nil
}
