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
	config     *config.Helm
	kubeClient clients.Kubernetes
	helmClient clients.Helm
	log        hclog.Logger
}

func NewHelm(c *config.Helm, kc clients.Kubernetes, hc clients.Helm, l hclog.Logger) *Helm {
	return &Helm{c, kc, hc, l}
}

func (h *Helm) Create() error {
	h.log.Info("Creating Helm chart", "ref", h.config.Name)

	_, destPath, _ := utils.CreateKubeConfigPath(h.config.ClusterRef.Name)

	// set the KubeConfig for the kubernetes client
	// this is used by the healthchecks
	err := h.kubeClient.SetConfig(destPath)
	if err != nil {
		xerrors.Errorf("unable to create Kubernetes client: %w", err)
	}

	err = h.helmClient.Create(destPath, h.config.Name, h.config.Chart, h.config.Values)
	if err != nil {
		return err
	}

	// we can now health check the install
	if h.config.HealthCheck != nil && len(h.config.HealthCheck.Pods) > 0 {
		to, err := time.ParseDuration(h.config.HealthCheck.Timeout)
		if err != nil {
			xerrors.Errorf("unable to parse healthcheck duration: %w", err)
		}

		err = h.kubeClient.HealthCheckPods(h.config.HealthCheck.Pods, to)
		if err != nil {
			xerrors.Errorf("healthcheck failed after helm chart setup: %w", err)
		}
	}

	// set the state
	h.config.State = config.Applied

	return nil
}

func (h *Helm) Destroy() error {
	h.log.Info("Destroy Helm chart", "ref", h.config.Name)
	return nil
}

func (h *Helm) Lookup() ([]string, error) {
	return []string{}, nil
}

// Config returns the config for the provider
func (c *Helm) Config() ConfigWrapper {
	return ConfigWrapper{"config.Helm", c.config}
}

// State returns the state from the config
func (c *Helm) State() config.State {
	return c.config.State
}

// SetState updates the state in the config
func (c *Helm) SetState(state config.State) {
	c.config.State = state
}
