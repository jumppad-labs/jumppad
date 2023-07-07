package resources

import "github.com/jumppad-labs/hclconfig/types"

type Progress struct {
	ID            string              `json:"id"`
	Prerequisites []string            `json:"prerequisites"`
	Conditions    []ProgressCondition `json:"conditions"`
	Status        string              `json:"status"`
}

type ProgressCondition struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type Validation struct {
	ID         string                `json:"id"`
	Conditions []ValidationCondition `json:"conditions"`
}

type ValidationCondition struct {
	ID               string `json:"id"`
	Check            string `json:"check"`
	Solve            string `json:"solve,omitempty"`
	FailureMessage   string `json:"failure_message"`
	SuccessMessage   string `json:"success_message,omitempty"`
	Target           string `json:"target,omitempty"`
	User             string `json:"user,omitempty"`
	WorkingDirectory string `json:"working_directory,omitempty"`
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
	cfg, err := LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(t.ID)
		if r != nil {
			kstate := r.(*Task)
			t.Progress = kstate.Progress
			t.Validation = kstate.Validation
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
