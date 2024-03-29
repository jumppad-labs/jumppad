package null

import (
	"context"
	"fmt"

	"github.com/jumppad-labs/hclconfig/types"
	sdk "github.com/jumppad-labs/plugin-sdk"
)

var _ sdk.Provider = &Provider{}

// Null is a noop provider
type Provider struct {
	config types.Resource
	log    sdk.Logger
}

func (p *Provider) Init(cfg types.Resource, l sdk.Logger) error {
	p.config = cfg
	p.log = l

	return nil
}

func (p *Provider) Create(ctx context.Context) error {
	p.log.Info(fmt.Sprintf("Creating %s", p.config.Metadata().Type), "ref", p.config.Metadata().ID)
	return nil
}

func (p *Provider) Destroy(ctx context.Context, force bool) error {
	return nil
}

func (p *Provider) Lookup() ([]string, error) {
	return nil, nil
}

func (p *Provider) Refresh(ctx context.Context) error {
	return nil
}

func (p *Provider) Changed() (bool, error) {
	return false, nil
}
