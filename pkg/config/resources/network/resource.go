package network

import (
	"github.com/jumppad-labs/hclconfig/types"
)

// TypeNetwork is the string resource type for Network resources
const TypeNetwork string = "network"

const (
	DefaultNetworkID     string = "resource.network.default"
	DefaultNetworkName   string = "jumppad"
	DefaultNetworkSubnet string = "10.0.10.0/24"
)

// Network defines a Docker network
type Network struct {
	// embedded type holding name, etc
	types.ResourceBase `hcl:",remain"`

	Subnet     string `hcl:"subnet" json:"subnet"`
	EnableIPv6 bool   `hcl:"enable_ipv6,optional" json:"enable_ipv6"`
}
