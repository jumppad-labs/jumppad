package resources

import "github.com/shipyard-run/hclconfig/types"

// TypeDocs is the resource string for a Docs resource
const TypeDocs string = "docs"

// Docs allows the running of a Docusaurus container which can be used for
// online tutorials or documentation
type Docs struct {
	types.ResourceMetadata `hcl:",remain"`

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	Networks []NetworkAttachment `hcl:"network,block" json:"networks,omitempty"` // Attach to the correct network // only when Image is specified

	Image *Image `hcl:"image,block" json:"image,omitempty"` // image to use for the container

	Path           string `hcl:"path" json:"path"`
	Port           int    `hcl:"port" json:"port"`
	LiveReloadPort int    `hcl:"live_reload_port,optional" json:"live_reload_port,omitempty"`
	OpenInBrowser  bool   `hcl:"open_in_browser,optional" json:"open_in_browser"` // When a host port is defined open the location in a browser

	IndexTitle string   `hcl:"index_title,optional" json:"index_title"`
	IndexPages []string `hcl:"index_pages,optional" json:"index_pages,omitempty"`

	// Output parameters

	// FQDN is the fully qualified domain name for the container, this can be used
	// to access the container from other sources
	FQDN string `hcl:"fqdn,optional" json:"fqdn,omitempty"`
}

func (d *Docs) Process() error {
	d.Path = ensureAbsolute(d.Path, d.File)

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
