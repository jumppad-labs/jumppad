package resources

// Image defines a docker image which will be pushed to the clusters Docker
// registry
type Image struct {
	Name string `hcl:"name" json:"name"`
	// Username is the Docker registry user to use for private repositories
	Username string `hcl:"username,optional" json:"username,omitempty"`
	// Password is the Docker registry password to use for private repositories
	Password string `hcl:"password,optional" json:"password,omitempty"`

	// output

	// ID is the unique identifier for the image, this is independent of tag
	// and changes each time the image is built. An image that has been tagged
	// multiple times also shares the same ID.
	ID string `hcl:"id,optional" json:"id,omitempty"`
}
