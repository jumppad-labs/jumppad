package terraform

import (
	"path"
	"strings"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	ctypes "github.com/jumppad-labs/jumppad/pkg/config/resources/container"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/zclconf/go-cty/cty"
)

// TypeTerraform is the resource string for a Terraform resource
const TypeTerraform string = "terraform"

// ExecRemote allows commands to be executed in remote containers
type Terraform struct {
	types.ResourceMetadata `hcl:",remain"`

	Networks []ctypes.NetworkAttachment `hcl:"network,block" json:"networks,omitempty"` // Attach to the correct network // only when Image is specified

	Source           string            `hcl:"source" json:"source"`                                          // Source directory containing Terraform config
	Version          string            `hcl:"version,optional" json:"version,omitempty"`                     // Version of terraform to use
	WorkingDirectory string            `hcl:"working_directory,optional" json:"working_directory,omitempty"` // Working directory to run terraform commands
	Environment      map[string]string `hcl:"environment,optional" json:"environment,omitempty"`             // environment variables to set when starting the container
	Variables        cty.Value         `hcl:"variables,optional" json:"-"`                                   // variables to pass to terraform

	// Computed values

	Output         cty.Value `hcl:"output,optional"`                                           // output values returned from Terraform
	SourceChecksum string    `hcl:"source_checksum,optional" json:"source_checksum,omitempty"` // checksum of the source directory
	ApplyOutput    string    `hcl:"apply_output,optional"`                                     // output from the terraform apply
}

func (t *Terraform) Process() error {
	// make sure mount paths are absolute
	t.Source = utils.EnsureAbsolute(t.Source, t.ResourceFile)

	if t.WorkingDirectory == "" {
		t.WorkingDirectory = "./"
	} else {
		if !strings.HasPrefix(t.WorkingDirectory, "/") {
			t.WorkingDirectory = "/" + t.WorkingDirectory
		}

		t.WorkingDirectory = path.Clean("." + t.WorkingDirectory)
	}

	// set the base version
	if t.Version == "" {
		t.Version = "1.16.2"
	}

	// restore the applyoutput from the state
	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(t.ResourceID)
		if r != nil {
			kstate := r.(*Terraform)
			t.ApplyOutput = kstate.ApplyOutput
			t.SourceChecksum = kstate.SourceChecksum
		}
	}

	return nil
}
