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
	c := config.New()
	cc := config.NewImageCache("tests")
	c.AddResource(cc)

	md := &mocks.MockContainerTasks{}
	hc := &mocks.MockHTTP{}

	md.On("CreateContainer", mock.Anything).Once().Return("abc", nil)
	md.On("CreateVolume", "images").Once().Return("images", nil)
	md.On("CopyFileToContainer", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	md.On("CopyFilesToVolume", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Once().Return(nil, nil)
	md.On("DetachNetwork", mock.Anything, mock.Anything).Return(nil)
	md.On("AttachNetwork", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	return cc, md, hc
}

func TestImageCacheCreateDoesNotCreateContainerWhenExists(t *testing.T) {
	cc, md, hc := setupImageCacheTests(t)

	c := NewImageCache(cc, md, hc, hclog.NewNullLogger())
	err := c.Create()
	assert.NoError(t, err)

	removeOn(&md.Mock, "FindContainerIDs")
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Once().Return([]string{"abc"}, nil)

	md.AssertNotCalled(t, "CreateContainer", "images")
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
	assert.Equal(t, "/cache", conf.Volumes[0].Destination)
	assert.Equal(t, "volume", conf.Volumes[0].Type)
}

func TestImageCacheCreateAddsEnvironmentVariables(t *testing.T) {
	cc, md, hc := setupImageCacheTests(t)

	c := NewImageCache(cc, md, hc, hclog.NewNullLogger())
	err := c.Create()
	assert.NoError(t, err)

	md.AssertCalled(t, "CreateContainer", mock.Anything)

	params := getCalls(&md.Mock, "CreateContainer")[0]
	conf := params.Arguments[0].(*config.Container)

	// check environment variables
	assert.Equal(t, conf.EnvVar["CA_KEY_FILE"], "/cache/ca/root.key")
	assert.Equal(t, conf.EnvVar["CA_CRT_FILE"], "/cache/ca/root.cert")
	assert.Equal(t, conf.EnvVar["DOCKER_MIRROR_CACHE"], "/cache/docker")
	assert.Equal(t, conf.EnvVar["ENABLE_MANIFEST_CACHE"], "true")
	assert.Equal(t, conf.EnvVar["REGISTRIES"], "k8s.gcr.io gcr.io asia.gcr.io eu.gcr.io us.gcr.io quay.io ghcr.io")
	assert.Equal(t, conf.EnvVar["ALLOW_PUSH"], "true")
}

func TestImageCacheCreateCopiesCerts(t *testing.T) {
	cc, md, hc := setupImageCacheTests(t)

	c := NewImageCache(cc, md, hc, hclog.NewNullLogger())
	err := c.Create()
	assert.NoError(t, err)

	md.AssertCalled(t, "CreateContainer", mock.Anything)

	// check copies certs
	md.AssertCalled(
		t,
		"CopyFilesToVolume",
		"images",
		[]string{
			filepath.Join(utils.CertsDir(""), "root.cert"),
			filepath.Join(utils.CertsDir(""), "root.key"),
		},
		"/ca",
		true,
	)
}

func TestImageCacheDetachesNetworks(t *testing.T) {
	net1 := config.NewNetwork("one")
	net2 := config.NewNetwork("two")

	cc, md, hc := setupImageCacheTests(t)
	cc.Networks = []string{"network.one", "network.two"}

	cc.Config.AddResource(net1)
	cc.Config.AddResource(net2)

	c := NewImageCache(cc, md, hc, hclog.NewNullLogger())
	err := c.Create()
	assert.NoError(t, err)

	md.AssertNumberOfCalls(t, "DetachNetwork", 2)
}

func TestImageCacheAttachesNetworks(t *testing.T) {
	net1 := config.NewNetwork("one")
	net2 := config.NewNetwork("two")

	cc, md, hc := setupImageCacheTests(t)
	cc.DependsOn = []string{"network.one", "network.two"}

	cc.Config.AddResource(net1)
	cc.Config.AddResource(net2)

	c := NewImageCache(cc, md, hc, hclog.NewNullLogger())
	err := c.Create()
	assert.NoError(t, err)

	md.AssertNumberOfCalls(t, "AttachNetwork", 2)
	md.AssertNumberOfCalls(t, "DetachNetwork", 0)
}
