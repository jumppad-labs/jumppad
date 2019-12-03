package config

// Docs allows the running of a Docusaurus container which can be used for
// online tutorials or documentation
type Docs struct {
	Name  string
	Path  string `hcl:"path"`
	Index string `hcl:"index,optional"`
	Port  int    `hcl:"port"`
}
