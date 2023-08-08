package docs

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
)

type Progress struct {
	ID            string              `hcl:"id,optional" json:"id"`
	Prerequisites []string            `hcl:"prerequisite,optional" json:"prerequisites"`
	Conditions    []ProgressCondition `hcl:"condition,block" json:"conditions"`
	Status        string              `hcl:"status,optional" json:"status"`
}

type ProgressCondition struct {
	ID          string `hcl:"id,optional" json:"id"`
	Description string `hcl:"description,optional" json:"description"`
	Status      string `hcl:"status,optional" json:"status"`
}

type Validation struct {
	ID         string                `hcl:"id,optional" json:"id"`
	Conditions []ValidationCondition `hcl:"condition,block" json:"conditions"`
}

type ValidationCondition struct {
	ID               string `hcl:"id,optional" json:"id"`
	Check            string `hcl:"check,optional" json:"check"`
	Solve            string `hcl:"solve,optional" json:"solve"`
	FailureMessage   string `hcl:"failure_message,optional" json:"failure_message"`
	SuccessMessage   string `hcl:"success_message,optional" json:"success_message,omitempty"`
	Target           string `hcl:"target,optional" json:"target,omitempty"`
	User             string `hcl:"user,optional" json:"user,omitempty"`
	WorkingDirectory string `hcl:"working_directory,optional" json:"working_directory,omitempty"`
}

const TypeTask string = "task"

type Task struct {
	types.ResourceMetadata `hcl:",remain"`

	Prerequisites []string    `hcl:"prerequisites,optional" json:"prerequisites"`
	Conditions    []Condition `hcl:"condition,block" json:"conditions"`

	// Output parameters
	Progress   Progress   `hcl:"progress,optional" json:"progress"`
	Validation Validation `hcl:"validation,optional" json:"validation"`
}

func (t *Task) Process() error {
	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(t.ID)
		if r != nil {
			state := r.(*Task)
			t.Progress = state.Progress
			t.Validation = state.Validation
		}
	}

	return nil
}

type Condition struct {
	Name             string `hcl:"id,label" json:"id"`
	Description      string `hcl:"description" json:"description"`
	Check            string `hcl:"check" json:"check"`
	Solve            string `hcl:"solve,optional" json:"solve,omitempty"`
	FailureMessage   string `hcl:"failure_message" json:"failure_message,omitempty"`
	SuccessMessage   string `hcl:"success_message,optional" json:"success_message,omitempty"`
	Target           string `hcl:"target,optional" json:"target,omitempty"`
	User             string `hcl:"user,optional" json:"user,omitempty"`
	WorkingDirectory string `hcl:"working_directory,optional" json:"working_directory,omitempty"`
}
