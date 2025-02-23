package nomad

import (
	"context"
	"fmt"
	"os"
	"time"

	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/nomad"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	sdk "github.com/jumppad-labs/plugin-sdk"
)

var _ sdk.Provider = &JobProvider{}

// NomadJob is a provider which enabled the creation and destruction
// of Nomad jobs
type JobProvider struct {
	config *NomadJob
	client nomad.Nomad
	log    sdk.Logger
}

func (p *JobProvider) Init(cfg htypes.Resource, l sdk.Logger) error {
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
func (p *JobProvider) Create(ctx context.Context) error {
	if ctx.Err() != nil {
		p.log.Debug("Skipping create, context cancelled", "ref", p.config.Meta.ID)
		return nil
	}

	p.log.Info("Create Nomad Job", "ref", p.config.Meta.ID, "files", p.config.Paths)

	nomadCluster := p.config.Cluster

	// load the config
	p.client.SetConfig(fmt.Sprintf("http://%s", nomadCluster.ExternalIP), nomadCluster.APIPort, nomadCluster.ClientNodes)

	err := p.client.Create(p.config.Paths)
	if err != nil {
		return fmt.Errorf("unable to create Nomad jobs: %w", err)
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
				if ctx.Err() != nil {
					return fmt.Errorf("context cancelled, unable to wait for job health")
				}

				if time.Since(st) >= dur {
					return fmt.Errorf("timeout waiting for job '%s' to start", j)
				}

				p.log.Debug("Checking health for", "ref", p.config.Meta.ID, "job", j)

				s, err := p.client.JobRunning(j)
				if err == nil && s {
					p.log.Debug("Health passed for", "ref", p.config.Meta.ID, "job", j)
					break
				}

				time.Sleep(1 * time.Second)
			}
		}

	}

	// set the checksums
	cs, err := p.generateChecksums()
	if err != nil {
		return fmt.Errorf("unable to generate checksums: %w", err)
	}

	p.config.JobChecksums = cs

	return nil
}

// Destroy the Nomad jobs defined by the config
func (p *JobProvider) Destroy(ctx context.Context, force bool) error {
	if ctx.Err() != nil {
		p.log.Debug("Skipping destroy, context cancelled", "ref", p.config.Meta.ID)
		return nil
	}

	p.log.Info("Destroy Nomad Job", "ref", p.config.Meta.ID)

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
func (p *JobProvider) Lookup() ([]string, error) {
	return nil, nil
}

func (p *JobProvider) Refresh(ctx context.Context) error {
	if ctx.Err() != nil {
		p.log.Debug("Skipping refresh, context cancelled", "ref", p.config.Meta.ID)
		return nil
	}

	cp, err := p.getChangedPaths()
	if err != nil {
		return err
	}

	if len(cp) < 1 {
		return nil
	}

	p.log.Info("Refresh Nomad Jobs", "ref", p.config.Meta.ID, "paths", cp)

	err = p.Destroy(ctx, false)
	if err != nil {
		return err
	}

	return p.Create(context.Background())
}

func (p *JobProvider) Changed() (bool, error) {
	cp, err := p.getChangedPaths()
	if err != nil {
		return false, err
	}

	if len(cp) > 0 {
		p.log.Debug("Nomad jobs changed, needs refresh", "ref", p.config.Meta.ID)
		return true, nil
	}

	return false, nil
}

// generateChecksums generates a sha256 checksum for each of the the paths
func (p *JobProvider) generateChecksums() ([]string, error) {
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
func (p *JobProvider) getChangedPaths() ([]string, error) {
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
