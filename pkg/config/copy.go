package config

// TypeCopy copies files from one location to another
const TypeCopy ResourceType = "copy"

// Docs allows the running of a Docusaurus container which can be used for
// online tutorials or documentation
type Copy struct {
	ResourceInfo `hcl:",remain" mapstructure:",squash"`

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	Source      string `hcl:"source" json:"source"`                              // Source file, folder or glob
	Destination string `hcl:"destination" json:"destination"`                    // Destination to write file or files to
	Permissions string `hcl:"permissions,optional" json:"permissions,omitempty"` // Permissions 0777 to set for written file

	CopiedFiles []string `json:"copied_files" mapstructure:"copied_files" state:"true"`
}

// NewCopy creates a new Copy config resource
func NewCopy(name string) *Copy {
	return &Copy{ResourceInfo: ResourceInfo{Name: name, Type: TypeCopy, Status: PendingCreation}}
}
