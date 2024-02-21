package blueprint

import "github.com/jumppad-labs/hclconfig/types"

// TypeContainer is the resource string for a Container resource
const TypeBlueprint string = "blueprint"

// Blueprint defines a stack blueprint for defining yard configs
type Blueprint struct {
	types.ResourceBase `hcl:",remain"`

	Title        string   `hcl:"title,optional" json:"title,omitempty"`
	Organization string   `hcl:"organization,optional" json:"organization,omitempty"`
	Author       string   `hcl:"author,optional" json:"author,omitempty"`
	Authors      []string `hcl:"authors,optional" json:"authors,omitempty"`
	Slug         string   `hcl:"slug,optional" json:"slug,omitempty"`
	Icon         string   `hcl:"icon,optional" json:"icon,omitempty"`
	Tags         []string `hcl:"tags,optional" json:"tags,omitempty"`
	Summary      string   `hcl:"summary,optional" json:"summary,omitempty"`
	Description  string   `hcl:"description,optional" json:"description,omitempty"`
}
