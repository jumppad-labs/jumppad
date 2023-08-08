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

	progress := Progress{
		ID:            p.config.ID,
		Prerequisites: p.config.Prerequisites,
		Status:        "locked",
	}

	if len(p.config.Prerequisites) == 0 {
		progress.Status = "unlocked"
		progress.Prerequisites = []string{}
	}

	validation := Validation{
		ID: p.config.ID,
	}

	for _, c := range p.config.Conditions {
		progress.Conditions = append(progress.Conditions, ProgressCondition{
			ID:          c.Name,
			Description: c.Description,
			Status:      "",
		})

		validation.Conditions = append(validation.Conditions, ValidationCondition{
			ID:               c.Name,
			Check:            fmt.Sprintf("/validation/%s/%s.check", p.config.ID, c.Name),
			Solve:            fmt.Sprintf("/validation/%s/%s.solve", p.config.ID, c.Name),
			FailureMessage:   c.FailureMessage,
			SuccessMessage:   c.SuccessMessage,
			Target:           c.Target,
			User:             c.User,
			WorkingDirectory: c.WorkingDirectory,
		})
	}

	p.config.Progress = progress
	p.config.Validation = validation

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
