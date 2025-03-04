package copy

import (
	"os"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// TypeCopy copies files from one location to another
const TypeCopy string = "copy"

/*
The copy resource allows files and directories to be copied from one location to another.

@resource
*/
type Copy struct {
	/*
	 embedded type holding name, etc

	 @ignore
	*/
	types.ResourceBase `hcl:",remain"`
	// Any explicit dependencies that this resource has
	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`
	// Source file, folder, url, git repo, etc
	Source string `hcl:"source" json:"source"`
	// Destination file or directory to write file or files to.
	Destination string `hcl:"destination" json:"destination"`
	// Unix file permissions to apply to coppied files and direcories.
	Permissions string `hcl:"permissions,optional" json:"permissions,omitempty" default:"0777"`
	/*
		List of the full paths of copied files.

		@computed
	*/
	CopiedFiles []string `hcl:"copied_files,optional" json:"copied_files"`
}

func (t *Copy) Process() error {
	// If the source is a local file, ensure it is absolute
	tempSource := utils.EnsureAbsolute(t.Source, t.Meta.File)
	if _, err := os.Stat(tempSource); err == nil {
		t.Source = tempSource
	}

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
