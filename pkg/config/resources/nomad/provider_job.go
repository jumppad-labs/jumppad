package nomad

import (
	"fmt"
	"os"
	"time"

	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/clients/nomad"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"golang.org/x/xerrors"
)

// NomadJob is a provider which enabled the creation and destruction
// of Nomad jobs
type NomadJobProvider struct {
	config *NomadJob
	client nomad.Nomad
	log    logger.Logger
}

func (p *NomadJobProvider) Init(cfg htypes.Resource, l logger.Logger) error {
	cli, err := clients.GenerateClients(l)
	if err != nil {
		return err
	}

	c, ok := cfg.(*NomadJob)
	if !ok {
		return fmt.Errorf("unable to initialize NomadJob provider, resource is not of type NomadJob")
	}

	p.config = c
	p.client = cli.Nomad
	p.log = l

	return nil
}

// Create the Nomad jobs defined by the config
func (p *NomadJobProvider) Create() error {
	p.log.Info("Create Nomad Job", "ref", p.config.ID, "files", p.config.Paths)

	nomadCluster := p.config.Cluster

	// load the config
	p.client.SetConfig(fmt.Sprintf("http://%s", nomadCluster.ExternalIP), nomadCluster.APIPort, nomadCluster.ClientNodes)

	err := p.client.Create(p.config.Paths)
	if err != nil {
		return xerrors.Errorf("Unable to create Nomad jobs: %w", err)
	}

	// if health check defined wait for jobs
	if p.config.HealthCheck != nil {
		st := time.Now()
		dur, err := time.ParseDuration(p.config.HealthCheck.Timeout)
		if err != nil {
			return err
		}

		for _, j := range p.config.HealthCheck.Jobs {
			for {
				if time.Since(st) >= dur {
					return xerrors.Errorf("timeout waiting for job '%s' to start", j)
				}

				p.log.Debug("Checking health for", "ref", p.config.ID, "job", j)

				s, err := p.client.JobRunning(j)
				if err == nil && s == true {
					p.log.Debug("Health passed for", "ref", p.config.ID, "job", j)
					break
				}

				time.Sleep(1 * time.Second)
			}
		}

	}

	// set the checksums
	cs, err := p.generateChecksums()
	if err != nil {
		return xerrors.Errorf("unable to generate checksums: %w", err)
	}

	p.config.JobChecksums = cs

	return nil
}

// Destroy the Nomad jobs defined by the config
func (p *NomadJobProvider) Destroy() error {
	p.log.Info("Destroy Nomad Job", "ref", p.config.ID)

	nomadCluster := p.config.Cluster

	// load the config
	p.client.SetConfig(fmt.Sprintf("http://%s", nomadCluster.ExternalIP), nomadCluster.APIPort, nomadCluster.ClientNodes)

	err := p.client.Stop(p.config.Paths)
	if err != nil {
		p.log.Error("Unable to destroy Nomad job", "error", err)
		return nil
	}

	return nil
}

// Lookup the Nomad jobs defined by the config
func (p *NomadJobProvider) Lookup() ([]string, error) {
	return nil, nil
}

func (p *NomadJobProvider) Refresh() error {
	cp, err := p.getChangedPaths()
	if err != nil {
		return err
	}

	if len(cp) < 1 {
		return nil
	}

	p.log.Info("Refresh Nomad Jobs", "ref", p.config.ID, "paths", cp)

	err = p.Destroy()
	if err != nil {
		return err
	}

	return p.Create()
}

func (p *NomadJobProvider) Changed() (bool, error) {
	cp, err := p.getChangedPaths()
	if err != nil {
		return false, err
	}

	if len(cp) > 0 {
		p.log.Debug("Nomad jobs changed, needs refresh", "ref", p.config.ID)
		return true, nil
	}

	return false, nil
}

// generateChecksums generates a sha256 checksum for each of the the paths
func (p *NomadJobProvider) generateChecksums() ([]string, error) {
	checksums := []string{}

	for _, p := range p.config.Paths {
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
func (p *NomadJobProvider) getChangedPaths() ([]string, error) {
	// get the checksums
	cs, err := p.generateChecksums()
	if err != nil {
		return nil, err
	}

	// if we have more checksums than previous assume everything has changed
	if len(p.config.JobChecksums) != len(cs) {
		return p.config.Paths, nil
	}

	// compare the checksums
	diff := []string{}
	for i, c := range p.config.JobChecksums {

		if c != cs[i] {
			diff = append(diff, p.config.Paths[i])
		}
	}

	return diff, nil
}

// /v1/jobs/parse
