package container

/*
A port stanza defines host to container communications

```hcl

	container {
	  port {
	    ...
	  }
	}

```
*/
type Port struct {
	/*
	 The local port in the container.

	 ```hcl
	 local = 80
	 ```
	*/
	Local string `hcl:"local" json:"local"`
	// @ignore
	Remote string `hcl:"remote,optional" json:"remote,omitempty"`
	/*
		The host port to map the local port to.

		```hcl
		host = 8080
		```
	*/
	Host string `hcl:"host,optional" json:"host,omitempty"`
	/*
		The protocol to use when exposing the port, can be "tcp", or "udp".

		```hcl
		protocol = "tcp"
		```
	*/
	Protocol string `hcl:"protocol,optional" json:"protocol,omitempty"`
	/*
		Should a browser window be automatically opened when this resource is created.
		Browser windows will open at the path specified by this property.

		@ignore
	*/
	OpenInBrowser string `hcl:"open_in_browser,optional" json:"open_in_browser" mapstructure:"open_in_browser"`
}

type Ports []Port

/*
A port_range stanza defines host to container communications by exposing a range of ports for the container.

```hcl

	container {
	  port_range {
	    ...
	  }
	}

```
*/
type PortRange struct {
	/*
		The port range to expose, e.g, `8080-8082` would expose the ports `8080`, `8081`, `8082`.

		```hcl
		range = "8080-8082"
		```
	*/
	Range string `hcl:"range" json:"local" mapstructure:"local"`
	/*
		Expose the port range on the host.

		```hcl
		enable_host = true
		```
	*/
	EnableHost bool `hcl:"enable_host,optional" json:"enable_host,omitempty" mapstructure:"enable_host"`
	/*
		The protocol to use when exposing the port, can be "tcp", or "udp".

		```hcl
		protocol = "tcp"
		```
	*/
	Protocol string `hcl:"protocol,optional" json:"protocol,omitempty"`
}

type PortRanges []PortRange
