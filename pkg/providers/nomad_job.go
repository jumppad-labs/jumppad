package providers

import (
	"fmt"
	"time"

	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"golang.org/x/xerrors"
)

// NomadJob is a provider which enabled the creation and destruction
// of Nomad jobs
type NomadJob struct {
	config *resources.NomadJob
	client clients.Nomad
	log    clients.Logger
}

// NewNomadJob creates a provider which can create and destroy Nomad jobs
func NewNomadJob(c *resources.NomadJob, hc clients.Nomad, l clients.Logger) *NomadJob {
	return &NomadJob{c, hc, l}
}

// Create the Nomad jobs defined by the config
func (n *NomadJob) Create() error {
	n.log.Info("Create Nomad Job", "ref", n.config.Name, "files", n.config.Paths)

	// find the cluster
	cc, err := n.config.ParentConfig.FindResource(n.config.Cluster)
	if err != nil {
		return err
	}

	nomadCluster := cc.(*resources.NomadCluster)

	// load the config
	n.client.SetConfig(fmt.Sprintf("http://%s", nomadCluster.ExternalIP), nomadCluster.APIPort, nomadCluster.ClientNodes)

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

		for _, j := range n.config.HealthCheck.Jobs {
			for {
				if time.Now().Sub(st) >= dur {
					return xerrors.Errorf("Timeout waiting for job '%s' to start", j)
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

	cc, err := n.config.ParentConfig.FindResource(n.config.Cluster)
	if err != nil {
		return err
	}

	nomadCluster := cc.(*resources.NomadCluster)

	// load the config
	n.client.SetConfig(fmt.Sprintf("http://%s", nomadCluster.ExternalIP), nomadCluster.APIPort, nomadCluster.ClientNodes)

	err = n.client.Stop(n.config.Paths)
	if err != nil {
		n.log.Error("Unable to destroy Nomad job", "error", err)
		return nil
	}

	return nil
}

// Lookup the Nomad jobs defined by the config
func (n *NomadJob) Lookup() ([]string, error) {
	return nil, nil
}

func (c *NomadJob) Refresh() error {
	c.log.Info("Refresh Nomad Job", "ref", c.config.Name)

	return nil
}

func (c *NomadJob) Changed() (bool, error) {
	c.log.Info("Checking changes", "ref", c.config.Name)

	return false, nil
}

// /v1/jobs/parse
