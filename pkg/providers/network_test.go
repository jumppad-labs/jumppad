package providers

import (
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

	return md, NewNetwork(c, md, hclog.Default())
}

func TestNetworkCreatesCorrectly(t *testing.T) {
	c := &config.Network{Name: "testnet", Subnet: "10.1.2.0/24"}
	md, p := setupNetworkTests(c)

	p.Create()

	md.AssertCalled(t, "NetworkCreate", mock.Anything, mock.Anything, mock.Anything)

	params := md.Calls[0].Arguments
	name := params[1].(string)
	nco := params[2].(types.NetworkCreate)

	assert.Equal(t, c.Name, name)
	assert.True(t, nco.Attachable)
	assert.Equal(t, "bridge", nco.Driver)
	assert.Equal(t, c.Subnet, nco.IPAM.Config[0].Subnet)
}
