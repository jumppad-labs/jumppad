package providers

import (
	"fmt"
	"os"
	"time"

	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/jumppad-labs/jumppad/pkg/utils"
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
				if time.Since(st) >= dur {
					return xerrors.Errorf("timeout waiting for job '%s' to start", j)
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

	// set the checksums
	cs, err := n.generateChecksums()
	if err != nil {
		return xerrors.Errorf("unable to generate checksums: %w", err)
	}

	n.config.JobChecksums = cs

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

func (n *NomadJob) Refresh() error {
	cp, err := n.getChangedPaths()
	if err != nil {
		return err
	}

	if len(cp) < 1 {
		return nil
	}

	n.log.Info("Refresh Nomad Jobs", "ref", n.config.ID, "paths", cp)

	err = n.Destroy()
	if err != nil {
		return err
	}

	return n.Create()
}

func (n *NomadJob) Changed() (bool, error) {
	cp, err := n.getChangedPaths()
	if err != nil {
		return false, err
	}

	if len(cp) > 0 {
		n.log.Debug("Nomad jobs changed, needs refresh", "ref", n.config.ID)
		return true, nil
	}

	return false, nil
}

// generateChecksums generates a sha256 checksum for each of the the paths
func (n *NomadJob) generateChecksums() ([]string, error) {
	checksums := []string{}

	for _, p := range n.config.Paths {
		f, err := os.Open(p)
		if err != nil {
			return nil, err
		}

		fi, err := f.Stat()
		if err != nil {
			return nil, err
		}

		var hash string

		if fi.IsDir() {
			hash, err = utils.HashDir(p)
		} else {
			hash, err = utils.HashFile(p)
		}

		if err != nil {
			return nil, err
		}

		checksums = append(checksums, hash)
	}

	return checksums, nil
}

// getChangedPaths returns the paths that have changed since the nomad jobs
// were last applied
func (n *NomadJob) getChangedPaths() ([]string, error) {
	// get the checksums
	cs, err := n.generateChecksums()
	if err != nil {
		return nil, err
	}

	// if we have more checksums than previous assume everything has changed
	if len(n.config.JobChecksums) != len(cs) {
		return n.config.Paths, nil
	}

	// compare the checksums
	diff := []string{}
	for i, c := range n.config.JobChecksums {

		if c != cs[i] {
			diff = append(diff, n.config.Paths[i])
		}
	}

	return diff, nil
}

// /v1/jobs/parse
