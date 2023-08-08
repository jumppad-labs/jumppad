package providers

import (
	"fmt"
	"strings"

	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
)

type Task struct {
	config *resources.Task
	log    clients.Logger
}

func NewTask(t *resources.Task, l clients.Logger) *Task {
	return &Task{t, l}
}

func (t *Task) Create() error {
	t.log.Info(fmt.Sprintf("Creating %s", strings.Title(string(t.config.Metadata().Type))), "ref", t.config.Metadata().Name)

	progress := resources.Progress{
		ID:            t.config.ID,
		Prerequisites: t.config.Prerequisites,
		Status:        "locked",
	}

	if len(t.config.Prerequisites) == 0 {
		progress.Status = "unlocked"
		progress.Prerequisites = []string{}
	}

	validation := resources.Validation{
		ID: t.config.ID,
	}

	for _, c := range t.config.Conditions {
		progress.Conditions = append(progress.Conditions, resources.ProgressCondition{
			ID:          c.Name,
			Description: c.Description,
			Status:      "",
		})

		validation.Conditions = append(validation.Conditions, resources.ValidationCondition{
			ID:               c.Name,
			Check:            fmt.Sprintf("/validation/%s/%s.check", t.config.ID, c.Name),
			Solve:            fmt.Sprintf("/validation/%s/%s.solve", t.config.ID, c.Name),
			FailureMessage:   c.FailureMessage,
			SuccessMessage:   c.SuccessMessage,
			Target:           c.Target,
			User:             c.User,
			WorkingDirectory: c.WorkingDirectory,
		})
	}

	t.config.Progress = progress
	t.config.Validation = validation

	return nil
}

func (t *Task) Destroy() error {
	return nil
}

func (t *Task) Lookup() ([]string, error) {
	return nil, nil
}

func (t *Task) Refresh() error {
	return nil
}

func (t *Task) Changed() (bool, error) {
	return false, nil
}
