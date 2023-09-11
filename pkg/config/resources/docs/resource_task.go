package docs

import (
	"github.com/jumppad-labs/hclconfig/types"
)

const TypeTask string = "task"

type Task struct {
	types.ResourceMetadata `hcl:",remain"`

	Prerequisites []string    `hcl:"prerequisites,optional" json:"prerequisites"`
	Config        *Config     `hcl:"config,block" json:"config,omitempty"`
	Conditions    []Condition `hcl:"condition,block" json:"conditions"`
	Status        string      `hcl:"status,optional" json:"status"`
}

type Condition struct {
	Name        string       `hcl:"id,label" json:"id"`
	Description string       `hcl:"description" json:"description"`
	Checks      []Validation `hcl:"check,block" json:"checks"`
	Solves      []Validation `hcl:"solve,block" json:"solves,omitempty"`
	Setups      []Validation `hcl:"setup,block" json:"setups,omitempty"`
	Cleanups    []Validation `hcl:"cleanup,block" json:"cleanups,omitempty"`
	Status      string       `hcl:"status,optional" json:"status"`
}

type Validation struct {
	Script         string `hcl:"script" json:"script"`
	FailureMessage string `hcl:"failure_message,optional" json:"failure_message,omitempty"`
	SuccessMessage string `hcl:"success_message,optional" json:"success_message,omitempty"`
	Config         `hcl:",remain"`
}

type Config struct {
	Timeout          int    `hcl:"timeout,optional" json:"timeout"`
	Target           string `hcl:"target,optional" json:"target,omitempty"`
	User             string `hcl:"user,optional" json:"user,omitempty"`
	Group            string `hcl:"group,optional" json:"group,omitempty"`
	WorkingDirectory string `hcl:"working_directory,optional" json:"working_directory,omitempty"`
}

func (t *Task) Process() error {
	// Set defaults
	if t.Config == nil {
		t.Config = &Config{}
	}

	if t.Config.Timeout == 0 {
		t.Config.Timeout = 30
	}

	if t.Config.User == "" {
		t.Config.User = "root"
	}

	if t.Config.WorkingDirectory == "" {
		t.Config.WorkingDirectory = "/"
	}

	// set overrides
	for i, condition := range t.Conditions {
		for j, check := range condition.Checks {
			if check.Timeout == 0 {
				t.Conditions[i].Checks[j].Timeout = t.Config.Timeout
			}

			if check.Target == "" {
				t.Conditions[i].Checks[j].Target = t.Config.Target
			}

			if check.User == "" {
				t.Conditions[i].Checks[j].User = t.Config.User
			}

			if check.Group == "" {
				t.Conditions[i].Checks[j].Group = t.Config.Group
			}

			if check.WorkingDirectory == "" {
				t.Conditions[i].Checks[j].WorkingDirectory = t.Config.WorkingDirectory
			}
		}

		for j, solve := range condition.Solves {
			if solve.Timeout == 0 {
				t.Conditions[i].Solves[j].Timeout = t.Config.Timeout
			}

			if solve.Target == "" {
				t.Conditions[i].Solves[j].Target = t.Config.Target
			}

			if solve.User == "" {
				t.Conditions[i].Solves[j].User = t.Config.User
			}

			if solve.Group == "" {
				t.Conditions[i].Solves[j].Group = t.Config.Group
			}

			if solve.WorkingDirectory == "" {
				t.Conditions[i].Solves[j].WorkingDirectory = t.Config.WorkingDirectory
			}
		}

		for j, setup := range condition.Setups {
			if setup.Timeout == 0 {
				t.Conditions[i].Setups[j].Timeout = t.Config.Timeout
			}

			if setup.Target == "" {
				t.Conditions[i].Setups[j].Target = t.Config.Target
			}

			if setup.User == "" {
				t.Conditions[i].Setups[j].User = t.Config.User
			}

			if setup.Group == "" {
				t.Conditions[i].Setups[j].Group = t.Config.Group
			}

			if setup.WorkingDirectory == "" {
				t.Conditions[i].Setups[j].WorkingDirectory = t.Config.WorkingDirectory
			}
		}
	}

	return nil
}
