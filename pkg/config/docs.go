package config

// TypeDocs is the resource string for a Docs resource
const TypeDocs ResourceType = "docs"

// Docs allows the running of a Docusaurus container which can be used for
// online tutorials or documentation
type Docs struct {
	ResourceInfo

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	Networks []NetworkAttachment `hcl:"network,block" json:"networks,omitempty"` // Attach to the correct network // only when Image is specified

	Image *Image `hcl:"image,block" json:"image,omitempty"` // image to use for the container

	Path          string `hcl:"path" json:"path"`
	Port          int    `hcl:"port" json:"port"`
	OpenInBrowser bool   `hcl:"open_in_browser,optional" json:"open_in_browser"`

	IndexTitle string   `hcl:"index_title,optional" json:"index_title"`
	IndexPages []string `hcl:"index_pages,optional" json:"index_pages,omitempty"`
}

// NewDocs creates a new Docs config resource
func NewDocs(name string) *Docs {
	return &Docs{ResourceInfo: ResourceInfo{Name: name, Type: TypeDocs, Status: PendingCreation}}
}
