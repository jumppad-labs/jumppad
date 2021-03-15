package providers

import (
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/stretchr/testify/mock"
	assert "github.com/stretchr/testify/require"
)

func setupImageCacheTests(t *testing.T) (*config.ImageCache, *mocks.MockContainerTasks, *mocks.MockHTTP) {
	cc := config.NewImageCache("tests")
	md := &mocks.MockContainerTasks{}
	hc := &mocks.MockHTTP{}

	md.On("CreateContainer", mock.Anything).Once().Return("abc", nil)
	md.On("CreateVolume", "images").Once().Return("", nil)
	md.On("CopyFileToContainer", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	return cc, md, hc
}

func TestImageCacheCreateCreatesVolume(t *testing.T) {
	cc, md, hc := setupImageCacheTests(t)

	c := NewImageCache(cc, md, hc, hclog.NewNullLogger())
	err := c.Create()
	assert.NoError(t, err)

	md.AssertCalled(t, "CreateVolume", "images")
}

func TestImageCacheCreateAddsVolumes(t *testing.T) {
	cc, md, hc := setupImageCacheTests(t)

	c := NewImageCache(cc, md, hc, hclog.NewNullLogger())
	err := c.Create()
	assert.NoError(t, err)

	md.AssertCalled(t, "CreateContainer", mock.Anything)

	params := getCalls(&md.Mock, "CreateContainer")[0]
	conf := params.Arguments[0].(*config.Container)

	// check volumes
	assert.Equal(t, utils.FQDNVolumeName("images"), conf.Volumes[0].Source)
	assert.Equal(t, "/docker_mirror_cache", conf.Volumes[0].Destination)
	assert.Equal(t, "volume", conf.Volumes[0].Type)
}

func TestImageCacheCreateCopiesCerts(t *testing.T) {
	cc, md, hc := setupImageCacheTests(t)

	c := NewImageCache(cc, md, hc, hclog.NewNullLogger())
	err := c.Create()
	assert.NoError(t, err)

	md.AssertCalled(t, "CreateContainer", mock.Anything)

	// check copies certs
	md.AssertCalled(t, "CopyFileToContainer", "abc", filepath.Join(utils.CertsDir(""), "root.cert"), "/ca/")
	md.AssertCalled(t, "CopyFileToContainer", "abc", filepath.Join(utils.CertsDir(""), "root.key"), "/ca/")
}
