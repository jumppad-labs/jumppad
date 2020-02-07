package providers

import (
	"fmt"
	"time"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"golang.org/x/xerrors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/kube"
)

type Helm struct {
	config     *config.Helm
	kubeClient clients.Kubernetes
	log        hclog.Logger
}

func NewHelm(c *config.Helm, kc clients.Kubernetes, l hclog.Logger) *Helm {
	return &Helm{c, kc, l}
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

	// set the kubeclient for Helm
	s := kube.GetConfig(destPath, "default", "default")
	cfg := &action.Configuration{}
	err = cfg.Init(s, "default", "", func(format string, v ...interface{}) {
		h.log.Debug("Helm debug message", "message", fmt.Sprintf(format, v...))
	})

	if err != nil {
		return xerrors.Errorf("unalbe to iniailize Helm: %w", err)
	}

	client := action.NewInstall(cfg)
	client.ReleaseName = h.config.Name
	client.Namespace = "default"

	settings := cli.EnvSettings{}
	p := getter.All(&settings)
	vo := values.Options{}

	// if we have an overriden values file set it
	if h.config.Values != "" {
		vo.ValueFiles = []string{h.config.Values}
	}

	vals, err := vo.MergeValues(p)
	if err != nil {
		return xerrors.Errorf("Error merging Helm values: %w", err)
	}

	cp, err := client.ChartPathOptions.LocateChart(h.config.Chart, &settings)
	if err != nil {
		return xerrors.Errorf("Error locating chart: %w", err)
	}

	chartRequested, err := loader.Load(cp)
	if err != nil {
		return xerrors.Errorf("Error loading chart: %w", err)
	}

	// merge values
	_, err = client.Run(chartRequested, vals)
	if err != nil {
		return xerrors.Errorf("Error running chart: %w", err)
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

	return nil
}

func (h *Helm) Destroy() error {
	h.log.Info("Destroy Helm chart", "ref", h.config.Name)
	return nil
}

func (h *Helm) Lookup() (string, error) {
	return "", nil
}
