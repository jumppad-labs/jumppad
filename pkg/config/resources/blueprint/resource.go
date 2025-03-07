package blueprint

import "github.com/jumppad-labs/hclconfig/types"

// TypeContainer is the resource string for a Container resource
const TypeBlueprint string = "blueprint"

/*
Blueprint defines a stack blueprint for defining yard configs

@resource
*/
type Blueprint struct {
	// @ignore
	types.ResourceBase `hcl:",remain"`

	// The title of the blueprint
	Title string `hcl:"title,optional" json:"title,omitempty"`
	// The namespace/organization the blueprint belongs to
	Organization string `hcl:"organization,optional" json:"organization,omitempty"`
	// The author of the blueprint
	Author string `hcl:"author,optional" json:"author,omitempty"`
	// The authors of the blueprint
	Authors []string `hcl:"authors,optional" json:"authors,omitempty"`
	// The slug of the blueprint
	Slug string `hcl:"slug,optional" json:"slug,omitempty"`
	// The url to an icon for the blueprint
	Icon string `hcl:"icon,optional" json:"icon,omitempty"`
	// A list of tags that describe the blueprint
	Tags []string `hcl:"tags,optional" json:"tags,omitempty"`
	// A summary of the description of the blueprint
	Summary string `hcl:"summary,optional" json:"summary,omitempty"`
	// A description of the blueprint
	Description string `hcl:"description,optional" json:"description,omitempty"`
}
