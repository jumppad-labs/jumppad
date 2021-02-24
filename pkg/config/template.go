package config

// TypeTemplate is the resource string for a Template resource
const TypeTemplate ResourceType = "template"

// Template allows the process of user defined templates
type Template struct {
	ResourceInfo

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	Source      string            `hcl:"source" json:"source"`                // Source template to be processed as string
	Destination string            `hcl:"destination" json:"destination"`      // Desintation filename to write
	Vars        map[string]string `hcl:"vars,optional" json:"vars,omitempty"` // Variables to be processed in the template
}

// NewTemplate creates a Template resource with the default values
func NewTemplate(name string) *Template {
	return &Template{ResourceInfo: ResourceInfo{Name: name, Type: TypeTemplate, Status: PendingCreation}}
}
