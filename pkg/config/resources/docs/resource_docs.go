package docs

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	ctypes "github.com/jumppad-labs/jumppad/pkg/config/resources/container"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// TypeDocs is the resource string for a Docs resource
const TypeDocs string = "docs"

// Docs allows the running of a Docusaurus container which can be used for
// online tutorials or documentation
type Docs struct {
	types.ResourceBase `hcl:",remain"`

	Networks ctypes.NetworkAttachments `hcl:"network,block" json:"networks,omitempty"` // Attach to the correct network // only when Image is specified

	Image *ctypes.Image `hcl:"image,block" json:"image,omitempty"` // image to use for the container

	Content []Book `hcl:"content" json:"content"`

	Port          int  `hcl:"port,optional" json:"port"`
	OpenInBrowser bool `hcl:"open_in_browser,optional" json:"open_in_browser"` // When a host port is defined open the location in a browser

	Logo   Logo   `hcl:"logo,optional" json:"logo,omitempty"`
	Assets string `hcl:"assets,optional" json:"assets,omitempty"`

	// Output parameters

	// ContainerName is the fully qualified resource name for the container, this can be used
	// to access the container from other sources
	ContainerName string `hcl:"fqdn,optional" json:"fqdn,omitempty"`

	// ContentChecksum is the checksum of the content directory, this is used to determine if the
	// docs need to be recreated
	ContentChecksum string `hcl:"content_checksum,optional" json:"content_checksum,omitempty"`
}

type Logo struct {
	URL    string `hcl:"url" json:"url"`
	Width  int    `hcl:"width" json:"width"`
	Height int    `hcl:"height" json:"height"`
}

func (d *Docs) Process() error {
	// if port not set set port to 80
	if d.Port == 0 {
		d.Port = 80
	}

	if d.Assets != "" {
		d.Assets = utils.EnsureAbsolute(d.Assets, d.Meta.File)
	}

	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(d.Meta.ID)
		if r != nil {
			kstate := r.(*Docs)
			d.ContainerName = kstate.ContainerName
			d.ContentChecksum = kstate.ContentChecksum
		}
	}

	return nil
}
