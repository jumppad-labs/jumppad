package docs

import (
	"github.com/jumppad-labs/hclconfig/types"
)

const TypeTask string = "task"

type Task struct {
	types.ResourceMetadata `hcl:",remain"`

	Prerequisites []string    `hcl:"prerequisites,optional" json:"prerequisites"`
	Config        Config      `hcl:"config,block" json:"config,omitempty"`
	Conditions    []Condition `hcl:"condition,block" json:"conditions"`
	Status        string      `hcl:"status,optional" json:"status"`
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
	Status           string `hcl:"status,optional" json:"status"`
}

type Config struct {
	Target           string `hcl:"target,optional" json:"target,omitempty"`
	User             string `hcl:"user,optional" json:"user,omitempty"`
	WorkingDirectory string `hcl:"working_directory,optional" json:"working_directory,omitempty"`
}
