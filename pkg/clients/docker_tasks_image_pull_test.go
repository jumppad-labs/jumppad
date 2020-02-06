package clients

import (
	"encoding/base64"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupImagePullMocks() *mocks.MockDocker {
	md := &mocks.MockDocker{}
	md.On("ImageList", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	md.On("ImagePull", mock.Anything, mock.Anything, mock.Anything).Return(
		ioutil.NopCloser(strings.NewReader("hello world")),
		nil,
	)

	return md
}

func createImagePullConfig() (config.Image, *mocks.MockDocker) {
	ic := config.Image{
		Name: "consul:1.6.1",
	}

	return ic, setupImagePullMocks()
}

func setupImagePull(t *testing.T, cc config.Image, md *mocks.MockDocker, force bool) {
	p := NewDockerTasks(md, hclog.NewNullLogger())

	// create the container
	err := p.PullImage(cc, force)
	assert.NoError(t, err)

	return
}

func TestPullImageWhenNOTCached(t *testing.T) {
	cc, md := createImagePullConfig()
	setupImagePull(t, cc, md, false)

	// test calls list image with a canonical image reference
	args := filters.NewArgs(filters.KeyValuePair{Key: "reference", Value: cc.Name})
	md.AssertCalled(t, "ImageList", mock.Anything, types.ImageListOptions{Filters: args})

	// test pulls image replacing the short name with the canonical registry name
	md.AssertCalled(t, "ImagePull", mock.Anything, makeImageCanonical(cc.Name), types.ImagePullOptions{})
}

func TestPullImageWithCredentialsWhenNOTCached(t *testing.T) {
	cc, md := createImagePullConfig()
	cc.Username = "nicjackson"
	cc.Password = "S3cur1t11"

	setupImagePull(t, cc, md, false)

	// test calls list image with a canonical image reference
	args := filters.NewArgs(filters.KeyValuePair{Key: "reference", Value: cc.Name})
	md.AssertCalled(t, "ImageList", mock.Anything, types.ImageListOptions{Filters: args})

	// test pulls image replacing the short name with the canonical registry name
	// adding credentials to image pull
	ipo := types.ImagePullOptions{RegistryAuth: createRegistryAuth(cc.Username, cc.Password)}
	md.AssertCalled(t, "ImagePull", mock.Anything, makeImageCanonical(cc.Name), ipo)

}

func TestPullImageWithValidCredentials(t *testing.T) {
	cc, md := createImagePullConfig()
	cc.Username = "nicjackson"
	cc.Password = "S3cur1t11"

	setupImagePull(t, cc, md, false)

	ipo := getCalls(&md.Mock, "ImagePull")[0].Arguments[2].(types.ImagePullOptions)

	d, err := base64.StdEncoding.DecodeString(ipo.RegistryAuth)
	assert.NoError(t, err)
	assert.Equal(t, `{"Username": "nicjackson", "Password": "S3cur1t11"}`, string(d))
}

// validate the registry auth is in the correct format
func TestPullImageNothingWhenCached(t *testing.T) {
	cc, md := createImagePullConfig()

	// remove the default image list which returns 0 cached images
	removeOn(&md.Mock, "ImageList")
	md.On("ImageList", mock.Anything, mock.Anything, mock.Anything).Return([]types.ImageSummary{types.ImageSummary{}}, nil)

	setupImagePull(t, cc, md, false)

	md.AssertNotCalled(t, "ImagePull", mock.Anything, mock.Anything, mock.Anything)
}

func TestPullImageAlwaysWhenForce(t *testing.T) {
	cc, md := createImagePullConfig()

	setupImagePull(t, cc, md, true)

	md.AssertNotCalled(t, "ImageList", mock.Anything, mock.Anything)
	md.AssertCalled(t, "ImagePull", mock.Anything, mock.Anything, mock.Anything)
}
