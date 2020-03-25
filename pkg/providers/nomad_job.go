package providers

import (
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
)

// NomadJob is a provider which enabled the creation and destruction
// of Nomad jobs
type NomadJob struct {
	config *config.NomadJob
	client clients.HTTP
	log    hclog.Logger
}

// NewNomadJob creates a provider which can create and destroy Nomad jobs
func NewNomadJob(c *config.NomadJob, hc clients.HTTP, l hclog.Logger) *NomadJob {
	return &NomadJob{c, hc, l}
}

// Create the Nomad jobs defined by the config
func (n *NomadJob) Create() error {
	return nil
}

// Destroy the Nomad jobs defined by the config
func (n *NomadJob) Destroy() error {
	return nil
}

// Lookup the Nomad jobs defined by the config
func (n *NomadJob) Lookup() ([]string, error) {
	return nil, nil
}

// /v1/jobs/parse
