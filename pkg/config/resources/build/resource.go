package build

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// TypeBuild builds containers and other resources
const TypeBuild string = "build"

type Build struct {
	// embedded type holding name, etc
	types.ResourceMetadata `hcl:",remain"`

	Container BuildContainer `hcl:"container,block" json:"container"`

	// outputs

	// Image is the full local reference of the built image
	Image string `hcl:"image,optional" json:"image"`

	// Checksum is calculated from the Context files
	BuildChecksum string `hcl:"build_checksum,optional" json:"build_checksum,omitempty"`
}

type BuildContainer struct {
	DockerFile string            `hcl:"dockerfile,optional" json:"dockerfile,omitempty"` // Location of build file inside build context defaults to ./Dockerfile
	Context    string            `hcl:"context" json:"context"`                          // Path to build context
	Args       map[string]string `hcl:"args,optional" json:"args,omitempty"`             // Build args to pass  to the container
}

func (b *Build) Process() error {
	b.Container.Context = utils.EnsureAbsolute(b.Container.Context, b.File)

	cfg, err := resources.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(b.ID)
		if r != nil {
			kstate := r.(*Build)
			b.Image = kstate.Image

			// add the build checksum
			b.BuildChecksum = kstate.BuildChecksum
		}
	}

	return nil
}
