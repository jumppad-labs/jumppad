package config

// TypeDocs is the resource string for a Docs resource
const TypeDocs ResourceType = "docs"

// Docs allows the running of a Docusaurus container which can be used for
// online tutorials or documentation
type Docs struct {
	ResourceInfo `mapstructure:",squash"`

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	Networks []NetworkAttachment `hcl:"network,block" json:"networks,omitempty"` // Attach to the correct network // only when Image is specified

	Image *Image `hcl:"image,block" json:"image,omitempty"` // image to use for the container

	Path           string `hcl:"path" json:"path"`
	Port           int    `hcl:"port" json:"port"`
	LiveReloadPort int    `hcl:"live_reload_port,optional" json:"live_reload_port,omitempty" mapstructure:"live_reload_port"`
	OpenInBrowser  bool   `hcl:"open_in_browser,optional" json:"open_in_browser" mapstructure:"open_in_browser"` // When a host port is defined open the location in a browser

	IndexTitle string   `hcl:"index_title,optional" json:"index_title" mapstructure:"index_title"`
	IndexPages []string `hcl:"index_pages,optional" json:"index_pages,omitempty" mapstructure:"index_pages"`
}

// NewDocs creates a new Docs config resource
func NewDocs(name string) *Docs {
	return &Docs{ResourceInfo: ResourceInfo{Name: name, Type: TypeDocs, Status: PendingCreation}}
}
