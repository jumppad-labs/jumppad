package providers

import (
	"fmt"

	"github.com/jumppad-labs/hclconfig/types"
	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
)

// Null is a noop provider
type NullProvider struct {
	config types.Resource
	log    logger.Logger
}

func (p *NullProvider) Init(cfg htypes.Resource, l logger.Logger) error {
	p.config = cfg
	p.log = l

	return nil
}

func (p *NullProvider) Create() error {
	p.log.Info(fmt.Sprintf("Creating %s", p.config.Metadata().Type), "ref", p.config.Metadata().ID)
	return nil
}

func (p *NullProvider) Destroy() error {
	return nil
}

func (p *NullProvider) Lookup() ([]string, error) {
	return nil, nil
}

func (p *NullProvider) Refresh() error {
	return nil
}

func (p *NullProvider) Changed() (bool, error) {
	return false, nil
}
