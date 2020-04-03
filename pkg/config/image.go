package config

// Image defines a docker image which will be pushed to the clusters Docker
// registry
type Image struct {
	Name string `hcl:"name" json:"name"`
	// Username is the Docker registry user to use for private repositories
	Username string `hcl:"username,optional" json:"username,omitempty"`
	// Password is the Docker registry password to use for private repositories
	Password string `hcl:"password,optional" json:"password,omitempty"`
}
