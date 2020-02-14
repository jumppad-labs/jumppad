package providers

import (
	"fmt"
	"testing"

	"github.com/docker/docker/api/types"
	hclog "github.com/hashicorp/go-hclog"
	clients "github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupNetworkTests(c *config.Network) (*clients.MockDocker, *Network) {
	md := &clients.MockDocker{}
	md.On("NetworkCreate", mock.Anything, mock.Anything, mock.Anything).
		Return(types.NetworkCreateResponse{}, nil)
	md.On("NetworkList", mock.Anything, mock.Anything).Return(nil, nil)

	return md, NewNetwork(c, md, hclog.Default())
}

func TestNetworkCreatesCorrectly(t *testing.T) {
	c := config.NewNetwork("testnet")
	c.Subnet = "10.1.2.0/24"

	md, p := setupNetworkTests(c)

	p.Create()

	md.AssertCalled(t, "NetworkCreate", mock.Anything, mock.Anything, mock.Anything)

	params := md.Calls[1].Arguments
	name := params[1].(string)
	nco := params[2].(types.NetworkCreate)

	assert.Equal(t, c.Name, name)
	assert.True(t, nco.Attachable)
	assert.Equal(t, "bridge", nco.Driver)
	assert.Equal(t, c.Subnet, nco.IPAM.Config[0].Subnet)
}

func TestNetworkDoesNOTCreateWhenExists(t *testing.T) {
	c := config.NewNetwork("testnet")
	c.Subnet = "10.1.2.0/24"

	md, p := setupNetworkTests(c)
	removeOn(&md.Mock, "NetworkList")
	md.On("NetworkList", mock.Anything, mock.Anything).Return([]types.NetworkResource{types.NetworkResource{ID: "abc"}}, nil)

	p.Create()

	md.AssertNotCalled(t, "NetworkCreate", mock.Anything, mock.Anything, mock.Anything)
}

func TestLookupReturnsID(t *testing.T) {
	c := config.NewNetwork("testnet")
	c.Subnet = "10.1.2.0/24"

	md, p := setupNetworkTests(c)
	removeOn(&md.Mock, "NetworkList")
	md.On("NetworkList", mock.Anything, mock.Anything).Return([]types.NetworkResource{types.NetworkResource{ID: "abc"}}, nil)

	ids, err := p.Lookup()
	assert.NoError(t, err)
	assert.Equal(t, "abc", ids[0])
}

func TestLookupFailReturnsError(t *testing.T) {
	c := config.NewNetwork("testnet")
	c.Subnet = "10.1.2.0/24"
	
	md, p := setupNetworkTests(c)
	removeOn(&md.Mock, "NetworkList")
	md.On("NetworkList", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("boom"))

	_, err := p.Lookup()
	assert.Error(t, err)
}
