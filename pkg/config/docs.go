package config

// TypeDocs is the resource string for a Docs resource
const TypeDocs ResourceType = "docs"

// Docs allows the running of a Docusaurus container which can be used for
// online tutorials or documentation
type Docs struct {
	ResourceInfo

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	Networks []NetworkAttachment `hcl:"network,block" json:"networks,omitempty"` // Attach to the correct network // only when Image is specified

	Path  string `hcl:"path"`
	Index string `hcl:"index,optional"`
	Port  int    `hcl:"port"`

	Image *Image `hcl:"image,block"` // image to use for the container
}

// NewDocs creates a new Docs config resource
func NewDocs(name string) *Docs {
	return &Docs{ResourceInfo: ResourceInfo{Name: name, Type: TypeDocs, Status: PendingCreation}}
}
