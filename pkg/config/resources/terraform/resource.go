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

/*
ExecRemote allows commands to be executed in remote containers

@resource
*/
type Terraform struct {
	/*
	 embedded type holding name, etc

	 @ignore
	*/
	types.ResourceBase `hcl:",remain"`
	/*
		Network attaches the container to an existing network defined in a separate stanza.
		This block can be specified multiple times to attach the container to multiple networks.

		@type container.NetworkAttachment
	*/
	Networks []ctypes.NetworkAttachment `hcl:"network,block" json:"networks,omitempty"`
	// The source directory containing the Terraform config to provision.
	Source string `hcl:"source" json:"source"`
	// The version of Terraform to use.
	Version string `hcl:"version,optional" json:"version,omitempty"`
	// The working directory to run the Terraform commands.
	WorkingDirectory string `hcl:"working_directory,optional" json:"working_directory,omitempty"`
	// Environment variables to set.
	Environment map[string]string `hcl:"environment,optional" json:"environment,omitempty"`
	/*
		Terraform variables to pass to Terraform.

		@type map[string]any
	*/
	Variables cty.Value `hcl:"variables,optional" json:"-"`
	/*
		A volume allows you to specify a local volume which is mounted to the container when it is created.
		This stanza can be specified multiple times.

		@example
		```
		volume {
			source      = "./"
			destination = "/files"
		}
		```
	*/
	Volumes []ctypes.Volume `hcl:"volume,block" json:"volumes,omitempty"`
	/*
		Any outputs defined in the Terraform configuration will be exposed as output
		values on the Terraform resource.

		@computed
	*/
	Output cty.Value `hcl:"output,optional"`
	/*
		checksum of the source directory

		@ignore
		@computed
	*/
	SourceChecksum string `hcl:"source_checksum,optional" json:"source_checksum,omitempty"`
	/*
		Console output from the Terraform apply.

		@computed
	*/
	ApplyOutput string `hcl:"apply_output,optional"`
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
