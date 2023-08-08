package random

import (
	"fmt"
	"math/rand"

	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
)

// RandomNumber is a random number provider
type RandomNumberProvider struct {
	config *RandomNumber
	log    logger.Logger
}

func (p *RandomNumberProvider) Init(cfg htypes.Resource, l logger.Logger) error {
	c, ok := cfg.(*RandomNumber)
	if !ok {
		return fmt.Errorf("unable to initialize RandomNumber provider, resource is not of type RandomNumber")
	}

	p.config = c
	p.log = l

	return nil
}

func (p *RandomNumberProvider) Create() error {
	p.log.Info("Creating random number", "ref", p.config.ID)

	number := rand.Intn(p.config.Maximum-p.config.Minimum) + p.config.Minimum
	p.log.Debug("Generated random number", "ref", p.config.ID, "number", number)

	p.config.Value = number

	return nil
}

func (p *RandomNumberProvider) Destroy() error {
	return nil
}

func (p *RandomNumberProvider) Lookup() ([]string, error) {
	return nil, nil
}

func (p *RandomNumberProvider) Refresh() error {
	return nil
}

func (p *RandomNumberProvider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.ID)

	return false, nil
}
