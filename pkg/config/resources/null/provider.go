package null

import (
	"fmt"

	"github.com/jumppad-labs/hclconfig/types"
	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
)

// Null is a noop provider
type Provider struct {
	config types.Resource
	log    logger.Logger
}

func (p *Provider) Init(cfg htypes.Resource, l logger.Logger) error {
	p.config = cfg
	p.log = l

	return nil
}

func (p *Provider) Create() error {
	p.log.Info(fmt.Sprintf("Creating %s", p.config.Metadata().Type), "ref", p.config.Metadata().ID)
	return nil
}

func (p *Provider) Destroy() error {
	return nil
}

func (p *Provider) Lookup() ([]string, error) {
	return nil, nil
}

func (p *Provider) Refresh() error {
	return nil
}

func (p *Provider) Changed() (bool, error) {
	return false, nil
}
