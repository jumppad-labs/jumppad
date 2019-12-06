package providers

import (
	"fmt"
	"os"

	"github.com/shipyard-run/cli/pkg/config"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/kube"
)

type Helm struct {
	config *config.Helm
}

func NewHelm(c *config.Helm) *Helm {
	return &Helm{c}
}

func (h *Helm) Create() error {
	// TODO refactor out in to a generic function
	destDir := fmt.Sprintf("%s/.shipyard/config/%s", os.Getenv("HOME"), h.config.ClusterRef.Name)
	destPath := fmt.Sprintf("%s/kubeconfig.yaml", destDir)

	s := kube.GetConfig(destPath, "default", "default")
	cfg := &action.Configuration{}
	if err := cfg.Init(s, "default", "", debug); err != nil {
		debug("%+v", err)
		os.Exit(1)
	}

	client := action.NewInstall(cfg)
	client.ReleaseName = h.config.Name
	client.Namespace = "default"

	settings := cli.EnvSettings{}
	p := getter.All(&settings)
	vo := values.Options{
		// ValueFiles: []string{""}
	}
	vals, err := vo.MergeValues(p)
	if err != nil {
		return err
	}

	cp, err := client.ChartPathOptions.LocateChart(h.config.Chart, &settings)
	if err != nil {
		return err
	}

	chartRequested, err := loader.Load(cp)
	if err != nil {
		return err
	}

	// merge values
	_, err = client.Run(chartRequested, vals)
	if err != nil {
		return err
	}

	return nil
}

func (h *Helm) Destroy() error {
	return nil
}

func (h *Helm) Lookup() (string, error) {
	return "", nil
}

func debug(format string, v ...interface{}) {
	format = fmt.Sprintf("[debug] %s\n", format)
	fmt.Printf(format, v...)
}
