package clients

import (
	"fmt"
	"sync"

	"github.com/hashicorp/go-hclog"
	"golang.org/x/xerrors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/kube"
)

type Helm interface {
	Create(kubeConfig, name, chartPath, valuesPath string) error
}

type HelmImpl struct {
	log  hclog.Logger
	lock sync.Mutex
}

func NewHelm(l hclog.Logger) Helm {
	return &HelmImpl{l, sync.Mutex{}}
}

func (h *HelmImpl) Create(kubeConfig, name, chartPath, valuesPath string) error {
	// set the kubeclient for Helm

	// possible race condition on GetConfig so aquire a lock
	h.lock.Lock()
	defer h.lock.Unlock()

	s := kube.GetConfig(kubeConfig, "default", "default")
	cfg := &action.Configuration{}
	err := cfg.Init(s, "default", "", func(format string, v ...interface{}) {
		h.log.Debug("Helm debug message", "message", fmt.Sprintf(format, v...))
	})

	if err != nil {
		return xerrors.Errorf("unalbe to iniailize Helm: %w", err)
	}

	client := action.NewInstall(cfg)
	client.ReleaseName = name
	client.Namespace = "default"

	settings := cli.EnvSettings{}
	p := getter.All(&settings)
	vo := values.Options{}

	// if we have an overriden values file set it
	if valuesPath != "" {
		vo.ValueFiles = []string{valuesPath}
	}

	vals, err := vo.MergeValues(p)
	if err != nil {
		return xerrors.Errorf("Error merging Helm values: %w", err)
	}

	cp, err := client.ChartPathOptions.LocateChart(chartPath, &settings)
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

	return nil
}
