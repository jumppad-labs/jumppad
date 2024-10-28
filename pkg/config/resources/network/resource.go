package network

import (
	"fmt"

	"github.com/jumppad-labs/hclconfig/types"
)

// TypeNetwork is the string resource type for Network resources
const TypeNetwork string = "network"

// Network defines a Docker network
type Network struct {
	// embedded type holding name, etc
	types.ResourceBase `hcl:",remain"`

	Subnet     string `hcl:"subnet" json:"subnet"`
	EnableIPv6 bool   `hcl:"enable_ipv6,optional" json:"enable_ipv6"`
}

func (c *Network) Parse(conf types.Findable) error {
	// do any other networks with this name exist?
	nets, err := conf.FindResourcesByType(TypeNetwork)
	if err != nil {
		return err
	}

	for _, n := range nets {
		if n.Metadata().Name == c.Meta.Name && n.Metadata().ID != c.Meta.ID {
			return fmt.Errorf("a network named '%s' is already defined by the resource '%s'", c.Meta.Name, n.Metadata().ID)
		}
	}

	return nil
}

func (c *Network) Process() error {
	return nil
}
