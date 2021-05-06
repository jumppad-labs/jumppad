package providers

import (
	"time"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"golang.org/x/xerrors"
)

type Helm struct {
	config       *config.Helm
	kubeClient   clients.Kubernetes
	helmClient   clients.Helm
	getterClient clients.Getter
	log          hclog.Logger
}

// NewHelm creates a new Helm provider
func NewHelm(c *config.Helm, kc clients.Kubernetes, hc clients.Helm, g clients.Getter, l hclog.Logger) *Helm {
	return &Helm{c, kc, hc, g, l}
}

// Create implements the provider Create method
func (h *Helm) Create() error {
	h.log.Info("Creating Helm chart", "ref", h.config.Name)

	// get the target cluster
	kcPath, err := h.getKubeConfigPath()
	if err != nil {
		return err
	}

	// if the namespace is null set to default
	if h.config.Namespace == "" {
		h.config.Namespace = "default"
	}

	// is the source a helm repo which should be downloaded?
	if !utils.IsLocalFolder(h.config.Chart) {
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
	// this is used by the healthchecks
	h.log.Debug("Using Kubernetes config", "ref", h.config.Name, "path", kcPath)
	h.kubeClient, err = h.kubeClient.SetConfig(kcPath)
	if err != nil {
		return xerrors.Errorf("unable to create Kubernetes client: %w", err)
	}

	err = h.helmClient.Create(
		kcPath, h.config.ChartName,
		h.config.Namespace, h.config.CreateNamespace,
		h.config.Chart, h.config.Values, h.config.ValuesString)

	if err != nil {
		return err
	}

	// we can now health check the install
	if h.config.HealthCheck != nil && len(h.config.HealthCheck.Pods) > 0 {
		to, err := time.ParseDuration(h.config.HealthCheck.Timeout)
		if err != nil {
			return xerrors.Errorf("unable to parse healthcheck duration: %w", err)
		}

		err = h.kubeClient.HealthCheckPods(h.config.HealthCheck.Pods, to)
		if err != nil {
			return xerrors.Errorf("healthcheck failed after helm chart setup: %w", err)
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

	// get the target cluster
	h.helmClient.Destroy(kcPath, h.config.ChartName, h.config.Namespace)

	if err != nil {
		h.log.Debug("There was a problem destroying Helm chart, logging message but ignoring error", "ref", h.config.Name, "error", err)
	}

	return nil
}

// Lookup implements the provider Lookup method
func (h *Helm) Lookup() ([]string, error) {
	return []string{}, nil
}

func (h *Helm) getKubeConfigPath() (string, error) {
	target, err := h.config.FindDependentResource(h.config.Cluster)
	if err != nil {
		return "", xerrors.Errorf("Unable to find cluster: %w", err)
	}

	_, destPath, _ := utils.CreateKubeConfigPath(target.Info().Name)
	return destPath, nil
}
