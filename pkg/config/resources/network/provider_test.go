package network

import (
	"context"
	"fmt"
	"testing"

	dtypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/container/mocks"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/testutils"
	"github.com/stretchr/testify/mock"
	assert "github.com/stretchr/testify/require"
)

var bridgeNetwork = dtypes.NetworkResource{
	ID:   "bridge",
	Name: "bridge",
	IPAM: network.IPAM{
		Config: []network.IPAMConfig{network.IPAMConfig{Subnet: "10.8.2.0/24"}},
	},
}

func setupNetworkTests(t *testing.T, c *Network) (*mocks.Docker, *Provider) {
	md := &mocks.Docker{}
	md.On("NetworkCreate", mock.Anything, mock.Anything, mock.Anything).Return(dtypes.NetworkCreateResponse{}, nil)
	md.On("NetworkList", mock.Anything, mock.Anything).Return([]dtypes.NetworkResource{bridgeNetwork}, nil)

	return md, &Provider{
		config: c,
		client: md,
		log:    logger.NewTestLogger(t),
	}
}

func TestLookupReturnsID(t *testing.T) {
	c := &Network{
		ResourceBase: types.ResourceBase{Meta: types.Meta{Name: "testnetwork"}},
	}

	c.Subnet = "10.1.2.0/24"

	md, p := setupNetworkTests(t, c)
	testutils.RemoveOn(&md.Mock, "NetworkList")
	md.On("NetworkList", mock.Anything, mock.Anything).Return([]dtypes.NetworkResource{
		dtypes.NetworkResource{
			ID: "testnet",
			IPAM: network.IPAM{
				Config: []network.IPAMConfig{network.IPAMConfig{Subnet: "10.1.2.0/24"}},
			},
		},
		bridgeNetwork,
	}, nil)

	ids, err := p.Lookup()
	assert.NoError(t, err)
	assert.Equal(t, "testnet", ids[0])
}
func TestLookupFailReturnsError(t *testing.T) {
	c := &Network{
		ResourceBase: types.ResourceBase{Meta: types.Meta{Name: "testnetwork"}},
	}

	c.Subnet = "10.1.2.0/24"

	md, p := setupNetworkTests(t, c)
	testutils.RemoveOn(&md.Mock, "NetworkList")
	md.On("NetworkList", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("boom"))

	_, err := p.Lookup()
	assert.Error(t, err)
}
func TestNetworkCreatesCorrectly(t *testing.T) {
	c := &Network{
		ResourceBase: types.ResourceBase{Meta: types.Meta{Name: "testnetwork"}},
	}
	c.Subnet = "10.1.2.0/24"

	md, p := setupNetworkTests(t, c)

	err := p.Create(context.Background())

	assert.NoError(t, err)

	md.AssertCalled(t, "NetworkCreate", mock.Anything, mock.Anything, mock.Anything)

	params := md.Calls[1].Arguments
	name := params[1].(string)
	nco := params[2].(dtypes.NetworkCreate)

	assert.Equal(t, c.Meta.Name, name)
	assert.True(t, nco.Attachable)
	assert.Equal(t, "bridge", nco.Driver)
	assert.Equal(t, c.Subnet, nco.IPAM.Config[0].Subnet)
}

func TestNetworkCreatesNatWhenNoBridge(t *testing.T) {
	c := &Network{
		ResourceBase: types.ResourceBase{Meta: types.Meta{Name: "testnetwork"}},
	}
	c.Subnet = "10.1.2.0/24"

	md, p := setupNetworkTests(t, c)

	testutils.RemoveOn(&md.Mock, "NetworkList")
	testutils.RemoveOn(&md.Mock, "NetworkCreate")
	md.On("NetworkList", mock.Anything, mock.Anything).Return(nil, nil)
	md.On("NetworkCreate", mock.Anything, mock.Anything, mock.Anything).Once().Return(dtypes.NetworkCreateResponse{}, fmt.Errorf("boom"))
	md.On("NetworkCreate", mock.Anything, mock.Anything, mock.Anything).Once().Return(dtypes.NetworkCreateResponse{}, nil)

	p.Create(context.Background())

	md.AssertNumberOfCalls(t, "NetworkCreate", 2)
	params := md.Calls[2].Arguments
	nco := params[2].(dtypes.NetworkCreate)

	assert.Equal(t, "nat", nco.Driver)
}

func TestNetworkDoesNOTCreateWhenExists(t *testing.T) {
	c := &Network{
		ResourceBase: types.ResourceBase{Meta: types.Meta{Name: "testnetwork"}},
	}
	c.Subnet = "10.1.2.0/24"

	md, p := setupNetworkTests(t, c)
	testutils.RemoveOn(&md.Mock, "NetworkList")
	md.On("NetworkList", mock.Anything, mock.Anything).Return([]dtypes.NetworkResource{
		dtypes.NetworkResource{
			ID: "testnet",
			IPAM: network.IPAM{
				Config: []network.IPAMConfig{network.IPAMConfig{Subnet: "10.1.2.0/24"}},
			},
		}, bridgeNetwork,
	}, nil)

	p.Create(context.Background())

	md.AssertNotCalled(t, "NetworkCreate", mock.Anything, mock.Anything, mock.Anything)
}

func TestCreateWithCorrectNameAndDifferentSubnetReturnsError(t *testing.T) {
	c := &Network{
		ResourceBase: types.ResourceBase{Meta: types.Meta{Name: "testnetwork"}},
	}
	c.Subnet = "10.1.2.0/16"

	md, p := setupNetworkTests(t, c)
	testutils.RemoveOn(&md.Mock, "NetworkList")
	md.On("NetworkList", mock.Anything, mock.Anything).Return([]dtypes.NetworkResource{
		dtypes.NetworkResource{
			ID: "testnet",
			IPAM: network.IPAM{
				Config: []network.IPAMConfig{network.IPAMConfig{Subnet: "10.1.1.0/24"}},
			},
		}, bridgeNetwork,
	}, nil)

	err := p.Create(context.Background())
	assert.Error(t, err)
}

func TestCreateWithOverlappingSubnetReturnsError(t *testing.T) {
	c := &Network{
		ResourceBase: types.ResourceBase{Meta: types.Meta{Name: "testnetwork"}},
	}
	c.Subnet = "10.2.3.0/16"

	md, p := setupNetworkTests(t, c)
	testutils.RemoveOn(&md.Mock, "NetworkList")
	md.On("NetworkList", mock.Anything, mock.Anything).Return([]dtypes.NetworkResource{
		dtypes.NetworkResource{
			ID: "abc",
			IPAM: network.IPAM{
				Config: []network.IPAMConfig{network.IPAMConfig{Subnet: "10.2.0.0/24"}},
			},
		}, bridgeNetwork,
	}, nil)

	err := p.Create(context.Background())
	assert.Error(t, err)
}
