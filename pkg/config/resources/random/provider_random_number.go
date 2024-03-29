package random

import (
	"context"
	"fmt"
	"math/rand"

	htypes "github.com/jumppad-labs/hclconfig/types"
	sdk "github.com/jumppad-labs/plugin-sdk"
)

var _ sdk.Provider = &RandomNumberProvider{}

// RandomNumber is a random number provider
type RandomNumberProvider struct {
	config *RandomNumber
	log    sdk.Logger
}

func (p *RandomNumberProvider) Init(cfg htypes.Resource, l sdk.Logger) error {
	c, ok := cfg.(*RandomNumber)
	if !ok {
		return fmt.Errorf("unable to initialize RandomNumber provider, resource is not of type RandomNumber")
	}

	p.config = c
	p.log = l

	return nil
}

func (p *RandomNumberProvider) Create(ctx context.Context) error {
	p.log.Info("Creating random number", "ref", p.config.Meta.ID)

	number := rand.Intn(p.config.Maximum-p.config.Minimum) + p.config.Minimum
	p.log.Debug("Generated random number", "ref", p.config.Meta.ID, "number", number)

	p.config.Value = number

	return nil
}

func (p *RandomNumberProvider) Destroy(ctx context.Context, force bool) error {
	return nil
}

func (p *RandomNumberProvider) Lookup() ([]string, error) {
	return nil, nil
}

func (p *RandomNumberProvider) Refresh(ctx context.Context) error {
	return nil
}

func (p *RandomNumberProvider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.Meta.ID)

	return false, nil
}
