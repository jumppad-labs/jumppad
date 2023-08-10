package docs

import (
	"fmt"

	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
)

type ChapterProvider struct {
	config *Chapter
	log    logger.Logger
}

func (p *ChapterProvider) Init(cfg htypes.Resource, l logger.Logger) error {
	c, ok := cfg.(*Chapter)
	if !ok {
		return fmt.Errorf("unable to initialize Chapter provider, resource is not of type Chapter")
	}

	p.config = c
	p.log = l

	return nil
}

func (p *ChapterProvider) Create() error {
	p.log.Info(fmt.Sprintf("Creating %s", p.config.Type), "ref", p.config.ID)
	return nil
}

func (p *ChapterProvider) Destroy() error {
	return nil
}

func (p *ChapterProvider) Lookup() ([]string, error) {
	return nil, nil
}

func (p *ChapterProvider) Refresh() error {
	return nil
}

func (p *ChapterProvider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.ID)

	return false, nil
}
