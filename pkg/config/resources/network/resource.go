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
	types.ResourceMetadata `hcl:",remain"`

	Subnet string `hcl:"subnet" json:"subnet"`
}

func (c *Network) Parse(conf types.Findable) error {
	// do any other networks with this name exist?
	nets, err := conf.FindResourcesByType(TypeNetwork)
	if err != nil {
		return err
	}

	for _, n := range nets {
		if n.Metadata().ResourceName == c.ResourceName && n.Metadata().ResourceID != c.ResourceID {
			return fmt.Errorf("a network named '%s' is already defined by the resource '%s'", c.ResourceName, n.Metadata().ResourceID)
		}
	}

	return nil
}

func (c *Network) Process() error {
	return nil
}
