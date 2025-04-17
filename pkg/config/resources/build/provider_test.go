package build

import (
	"context"
	"fmt"
	"testing"

	"github.com/instruqt/jumppad/pkg/clients/container/mocks"
	"github.com/instruqt/jumppad/pkg/clients/container/types"
	"github.com/instruqt/jumppad/pkg/clients/logger"
	"github.com/instruqt/jumppad/pkg/config/resources/container"
	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupProvider(t *testing.T, b *Build) (*Provider, *mocks.ContainerTasks) {
	l := logger.NewTestLogger(t)

	mc := &mocks.ContainerTasks{}
	mc.On("BuildContainer", mock.Anything, true).Return("buildimage:abcde", nil)
	mc.On("FindImagesInLocalRegistry", fmt.Sprintf("jumppad.dev/localcache/%s", b.Meta.Name)).Return([]string{}, nil)
	mc.On("TagImage", mock.Anything, mock.Anything).Return(nil)
	mc.On("PushImage", mock.Anything).Return(nil)

	p := &Provider{
		config: b,
		client: mc,
		log:    l,
	}

	return p, mc
}

func TestCreatePushesToRegistry(t *testing.T) {
	b := &Build{
		ResourceBase: htypes.ResourceBase{Meta: htypes.Meta{Name: "test"}},
		Registries: []container.Image{
			container.Image{
				Name: "nicholasjackson/fake:latest",
			},
			container.Image{
				Name:     "authed/fake:latest",
				Username: "test",
				Password: "password",
			},
		},
	}

	p, mc := setupProvider(t, b)
	err := p.Create(context.Background())
	require.NoError(t, err)

	// ensure the image is tagged
	mc.AssertCalled(t, "TagImage", "buildimage:abcde", "nicholasjackson/fake:latest")
	mc.AssertCalled(t, "PushImage", types.Image{Name: "nicholasjackson/fake:latest", Username: "", Password: ""})
	mc.AssertCalled(t, "PushImage", types.Image{Name: "authed/fake:latest", Username: "test", Password: "password"})
}
