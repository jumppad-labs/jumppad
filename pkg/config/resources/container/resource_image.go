package container

/*
Image defines a Docker image used when creating this container.
An Image can be stored in a public or a private repository.

```hcl

	container {
	  image {
	    ...
	  }
	}

```

@example

	```
	image {
	  name = "myregistry.io/myimage:latest"
	  username = env("REGISTRY_USERNAME")
	  password = env("REGISTRY_PASSWORD")
	}
	```
*/
type Image struct {
	/*
		Name of the image to use when creating the container, can either be the full
		canonical name or short name for Docker official images. e.g. `consul:v1.6.1` or
		`docker.io/consul:v1.6.1`.

		```hcl
		name = "redis:latest"
		```
	*/
	Name string `hcl:"name" json:"name"`
	/*
		Username to use when connecting to a private image repository

		```hcl
		username = "my_username"
		```
	*/
	Username string `hcl:"username,optional" json:"username,omitempty"`
	/*
		Password to use when connecting to a private image repository, for both username
		and password interpolated environment variables can be used in place of static values.

		```hcl
		password = "my_password"
		```
	*/
	Password string `hcl:"password,optional" json:"password,omitempty"`

	/*
		ID is the unique identifier for the image, this is independent of tag
		and changes each time the image is built. An image that has been tagged
		multiple times also shares the same ID.
		ID string `hcl:"id,optional" json:"id,omitempty"`

		@computed
	*/
	ID string `hcl:"id,optional" json:"id,omitempty"`
}

type Images []Image
