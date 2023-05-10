package resources

import "github.com/shipyard-run/hclconfig/types"

// TypeContainer is the resource string for a Container resource
const TypeBlueprint string = "blueprint"

// Blueprint defines a stack blueprint for defining yard configs
type Blueprint struct {
	types.ResourceMetadata `hcl:",remain"`

	Title       string `hcl:"title,optional" json:"title,omitempty"`
	Author      string `hcl:"author,optional" json:"author,omitempty"`
	Slug        string `hcl:"slug,optional" json:"slug,omitempty"`
	Description string `hcl:"description,optional" json:"description,omitempty"`
}
