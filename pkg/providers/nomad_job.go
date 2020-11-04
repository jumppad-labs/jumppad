package providers

import (
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"golang.org/x/xerrors"
)

// NomadJob is a provider which enabled the creation and destruction
// of Nomad jobs
type NomadJob struct {
	config *config.NomadJob
	client clients.Nomad
	log    hclog.Logger
}

// NewNomadJob creates a provider which can create and destroy Nomad jobs
func NewNomadJob(c *config.NomadJob, hc clients.Nomad, l hclog.Logger) *NomadJob {
	return &NomadJob{c, hc, l}
}

// Create the Nomad jobs defined by the config
func (n *NomadJob) Create() error {
	n.log.Info("Create Nomad Job", "ref", n.config.Name, "files", n.config.Paths)

	// find the cluster
	cc, err := n.config.ResourceInfo.FindDependentResource(n.config.Cluster)
	if err != nil {
		return err
	}

	// load the config
	_, configPath := utils.CreateClusterConfigPath(cc.Info().Name)
	err = n.client.SetConfig(configPath)
	if err != nil {
		return xerrors.Errorf("Unable to load nomad config %s: %w", configPath, err)
	}

	err = n.client.Create(n.config.Paths)
	if err != nil {
		return xerrors.Errorf("Unable to create Nomad jobs: %w", err)
	}

	// if health check defined wait for jobs
	if n.config.HealthCheck != nil {
		st := time.Now()
		dur, err := time.ParseDuration(n.config.HealthCheck.Timeout)
		if err != nil {
			return err
		}

		for _, j := range n.config.HealthCheck.NomadJobs {
			for {
				if time.Now().Sub(st) >= dur {
					return xerrors.Errorf("Timeout waiting for health checks")
				}

				n.log.Debug("Checking health for", "ref", n.config.Name, "job", j)

				s, err := n.client.JobRunning(j)
				if err == nil && s == true {
					n.log.Debug("Health passed for", "ref", n.config.Name, "job", j)
					break
				}

				time.Sleep(1 * time.Second)
			}
		}

	}

	return nil
}

// Destroy the Nomad jobs defined by the config
func (n *NomadJob) Destroy() error {
	n.log.Info("Destroy Nomad Job", "ref", n.config.Name)

	// find the cluster
	cc, err := n.config.ResourceInfo.FindDependentResource(n.config.Cluster)
	if err != nil {
		return err
	}

	// load the config
	_, configPath := utils.CreateClusterConfigPath(cc.Info().Name)
	err = n.client.SetConfig(configPath)
	if err != nil {
		n.log.Error("Unable to load Nomad config", "config", configPath, "error", err)
		return nil
	}

	err = n.client.Stop(n.config.Paths)
	if err != nil {
		n.log.Error("Unable to destroy Nomad job", "config", configPath, "error", err)
		return nil
	}

	return nil
}

// Lookup the Nomad jobs defined by the config
func (n *NomadJob) Lookup() ([]string, error) {
	return nil, nil
}

// /v1/jobs/parse
