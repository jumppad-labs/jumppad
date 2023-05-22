package resources

import "github.com/shipyard-run/hclconfig/types"

// TypeDocs is the resource string for a Docs resource
const TypeDocs string = "docs"

// Docs allows the running of a Docusaurus container which can be used for
// online tutorials or documentation
type Docs struct {
	types.ResourceMetadata `hcl:",remain"`

	Networks []NetworkAttachment `hcl:"network,block" json:"networks,omitempty"` // Attach to the correct network // only when Image is specified

	Image *Image `hcl:"image,block" json:"image,omitempty"` // image to use for the container

	Path           string `hcl:"path" json:"path"`
	NavigationFile string `hcl:"navigation_file" json:"navigation_file"`
	Port           int    `hcl:"port,optional" json:"port"`
	OpenInBrowser  bool   `hcl:"open_in_browser,optional" json:"open_in_browser"` // When a host port is defined open the location in a browser

	// Output parameters

	// FQDN is the fully qualified domain name for the container, this can be used
	// to access the container from other sources
	FQDN string `hcl:"fqdn,optional" json:"fqdn,omitempty"`
}

func (d *Docs) Process() error {
	d.Path = ensureAbsolute(d.Path, d.File)
	d.NavigationFile = ensureAbsolute(d.NavigationFile, d.File)

	// if port not set set port to 80
	if d.Port == 0 {
		d.Port = 80
	}

	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(d.ID)
		if r != nil {
			kstate := r.(*Docs)
			d.FQDN = kstate.FQDN
		}
	}

	return nil
}
