package terraform

import (
	"path"
	"strings"

	"github.com/instruqt/jumppad/pkg/config"
	ctypes "github.com/instruqt/jumppad/pkg/config/resources/container"
	"github.com/instruqt/jumppad/pkg/utils"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/zclconf/go-cty/cty"
)

// TypeTerraform is the resource string for a Terraform resource
const TypeTerraform string = "terraform"

// ExecRemote allows commands to be executed in remote containers
type Terraform struct {
	types.ResourceBase `hcl:",remain"`

	Networks []ctypes.NetworkAttachment `hcl:"network,block" json:"networks,omitempty"` // Attach to the correct network // only when Image is specified

	Source           string            `hcl:"source" json:"source"`                                          // Source directory containing Terraform config
	Version          string            `hcl:"version,optional" json:"version,omitempty"`                     // Version of terraform to use
	WorkingDirectory string            `hcl:"working_directory,optional" json:"working_directory,omitempty"` // Working directory to run terraform commands
	Environment      map[string]string `hcl:"environment,optional" json:"environment,omitempty"`             // environment variables to set when starting the container
	Variables        cty.Value         `hcl:"variables,optional" json:"-"`                                   // variables to pass to terraform
	Volumes          []ctypes.Volume   `hcl:"volume,block" json:"volumes,omitempty"`                         // Volumes to attach to the container

	// Computed values

	Output         cty.Value `hcl:"output,optional"`                                           // output values returned from Terraform
	SourceChecksum string    `hcl:"source_checksum,optional" json:"source_checksum,omitempty"` // checksum of the source directory
	ApplyOutput    string    `hcl:"apply_output,optional"`                                     // output from the terraform apply
}

func (t *Terraform) Process() error {
	// make sure mount paths are absolute
	t.Source = utils.EnsureAbsolute(t.Source, t.Meta.File)

	if t.WorkingDirectory == "" {
		t.WorkingDirectory = "./"
	} else {
		if !strings.HasPrefix(t.WorkingDirectory, "/") {
			t.WorkingDirectory = "/" + t.WorkingDirectory
		}

		t.WorkingDirectory = path.Clean("." + t.WorkingDirectory)
	}

	// process volumes
	for i, v := range t.Volumes {
		// make sure mount paths are absolute when type is bind, unless this is the docker sock
		if v.Type == "" || v.Type == "bind" {
			t.Volumes[i].Source = utils.EnsureAbsolute(v.Source, t.Meta.File)
		}
	}

	// set the base version
	if t.Version == "" {
		t.Version = "1.9.8"
	}

	// restore the applyoutput from the state
	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(t.Meta.ID)
		if r != nil {
			kstate := r.(*Terraform)
			t.ApplyOutput = kstate.ApplyOutput
			t.SourceChecksum = kstate.SourceChecksum
		}
	}

	return nil
}
