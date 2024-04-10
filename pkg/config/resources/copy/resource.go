package copy

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// TypeCopy copies files from one location to another
const TypeCopy string = "copy"

// Docs allows the running of a Docusaurus container which can be used for
// online tutorials or documentation
type Copy struct {
	// embedded type holding name, etc
	types.ResourceBase `hcl:",remain"`

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	Source      string `hcl:"source" json:"source"`                              // Source file, folder or glob
	Destination string `hcl:"destination" json:"destination"`                    // Destination to write file or files to
	Permissions string `hcl:"permissions,optional" json:"permissions,omitempty"` // Permissions 0777 to set for written file

	// outputs
	CopiedFiles []string `hcl:"copied_files,optional" json:"copied_files"`
}

func (t *Copy) Process() error {
	t.Source = utils.EnsureAbsolute(t.Source, t.Meta.File)
	t.Destination = utils.EnsureAbsolute(t.Destination, t.Meta.File)

	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(t.Meta.ID)
		if r != nil {
			kstate := r.(*Copy)
			t.CopiedFiles = kstate.CopiedFiles
		}
	}

	return nil
}
