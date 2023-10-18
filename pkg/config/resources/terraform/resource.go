package terraform

import (
	"github.com/jumppad-labs/hclconfig/types"
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

	Volumes []ctypes.Volume `hcl:"volume,block" json:"volumes,omitempty"` // Volumes to mount to container

	WorkingDirectory string            `hcl:"working_directory,optional" json:"working_directory,omitempty"` // Working directory to execute commands
	Environment      map[string]string `hcl:"environment,optional" json:"environment,omitempty"`             // environment variables to set when starting the container
	Variables        cty.Value         `hcl:"variables,optional" json:"-"`                                   // variables to pass to terraform

	// Computed values

	Output cty.Value `hcl:"output,optional"` // value of the output
}

func (t *Terraform) Process() error {
	// process volumes
	// make sure mount paths are absolute
	for i, v := range t.Volumes {
		t.Volumes[i].Source = utils.EnsureAbsolute(v.Source, t.File)
	}

	t.Output = cty.EmptyObjectVal

	return nil
}
