package network

import (
	"github.com/jumppad-labs/hclconfig/types"
)

// TypeNetwork is the string resource type for Network resources
const TypeNetwork string = "network"

const (
	DefaultNetworkID     string = "resource.network.jumppad"
	DefaultNetworkName   string = "jumppad"
	DefaultNetworkSubnet string = "10.0.10.0/24"
)

/*
Network resources allow you to create isolated networks for your resources.
There is no limit to the number of Network resources you can create, the only limitation is that they must not have overlapping subnets.

```hcl

	resource "network" "name" {
	  ...
	}

```

@example
```

	resource "network" "local" {
	  subnet = "10.10.0.0/16"
	}

```

@resource
*/
type Network struct {
	/*
	 embedded type holding name, etc

	 @ignore
	*/
	types.ResourceBase `hcl:",remain"`

	/*
		Subnet to use for the network, must not overlap any other existing networks.

		```hcl
		subnet = "10.100.100.0/24"
		```
	*/
	Subnet string `hcl:"subnet" json:"subnet"`
	/*
		Enable IPv6 on the network

		```hcl
		enable_ipv6 = true
		```
	*/
	EnableIPv6 bool `hcl:"enable_ipv6,optional" json:"enable_ipv6"`
}
