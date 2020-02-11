package config

// Docs allows the running of a Docusaurus container which can be used for
// online tutorials or documentation
type Docs struct {
	Name   string
	State  State
	WANRef *Network

	Path  string `hcl:"path"`
	Index string `hcl:"index,optional"`
	Port  int    `hcl:"port"`

	Image *Image `hcl:"image,block"` // image to use for the container
}
