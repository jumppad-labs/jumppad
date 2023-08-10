package docs

import (
	"fmt"

	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
)

type TaskProvider struct {
	config *Task
	log    logger.Logger
}

func (p *TaskProvider) Init(cfg htypes.Resource, l logger.Logger) error {
	c, ok := cfg.(*Task)
	if !ok {
		return fmt.Errorf("unable to initialize Task provider, resource is not of type Task")
	}

	p.config = c
	p.log = l

	return nil
}

func (p *TaskProvider) Create() error {
	p.log.Info(fmt.Sprintf("Creating %s", p.config.Type), "ref", p.config.ID)
	return nil
}

func (p *TaskProvider) Destroy() error {
	return nil
}

func (p *TaskProvider) Lookup() ([]string, error) {
	return nil, nil
}

func (p *TaskProvider) Refresh() error {
	return nil
}

func (p *TaskProvider) Changed() (bool, error) {
	return false, nil
}
