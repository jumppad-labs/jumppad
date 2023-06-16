package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

type Task struct {
	config *resources.Task
	log    hclog.Logger
}

func NewTask(t *resources.Task, l hclog.Logger) *Task {
	return &Task{t, l}
}

func (t *Task) Create() error {
	t.log.Info(fmt.Sprintf("Creating %s", strings.Title(string(t.config.Metadata().Type))), "ref", t.config.Metadata().Name)

	checksPath := utils.GetLibraryFolder("checks", 0775)
	taskPath := filepath.Join(checksPath, t.config.ID)
	os.MkdirAll(taskPath, 0755)
	os.Chmod(taskPath, 0755)

	progress := resources.Progress{
		ID:            t.config.ID,
		Prerequisites: t.config.Prerequisites,
		Status:        "locked",
	}

	if len(t.config.Prerequisites) == 0 {
		progress.Status = "unlocked"
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

		checkPath := filepath.Join(taskPath, c.Name)

		err := os.WriteFile(checkPath, []byte(c.Check), 0755)
		if err != nil {
			return fmt.Errorf("Unable to write check %s to disk at %s", c.Name, taskPath)
		}

		validation.Conditions = append(validation.Conditions, resources.ValidationCondition{
			ID:               c.Name,
			Check:            filepath.Join("/checks", t.config.ID, c.Name),
			Solve:            c.Solve,
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
