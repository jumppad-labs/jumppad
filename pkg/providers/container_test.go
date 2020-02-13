package providers

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestContainerCreatesSuccessfully(t *testing.T) {
	cc := config.NewContainer("tests")
	md := &mocks.MockContainerTasks{}
	c := NewContainer(*cc, md, hclog.NewNullLogger())

	// check pulls image before creating container
	md.On("PullImage", cc.Image, false).Once().Return(nil)

	// check calls CreateContainer with the config
	md.On("CreateContainer", cc).Once().Return("", nil)

	err := c.Create()
	assert.NoError(t, err)
}

func TestContainerDoesNOTCreateWhenPullImageFail(t *testing.T) {
	cc := config.NewContainer("tests")
	md := &mocks.MockContainerTasks{}
	c := NewContainer(*cc, md, hclog.NewNullLogger())

	// check pulls image before creating container and return an erro
	imageErr := fmt.Errorf("Unable to pull image")
	md.On("PullImage", cc.Image, false).Once().Return(imageErr)

	// check does not call CreateContainer with the config
	md.On("CreateContainer", cc).Times(0)

	err := c.Create()
	assert.Equal(t, imageErr, err)
}

func TestContainerDestroysCorrectlyWhenContainerExists(t *testing.T) {
	cc := config.NewContainer("tests")
	cc.Networks = []config.NetworkAttachment{config.NetworkAttachment{ Name: "cloud" }}
	md := &mocks.MockContainerTasks{}
	c := NewContainer(*cc, md, hclog.NewNullLogger())

	md.On("FindContainerIDs", cc.Name, cc.Type).Return([]string{"abc"}, nil)
	md.On("RemoveContainer", "abc").Return(nil)

	err := c.Destroy()
	assert.NoError(t, err)
}

func TestContainerDoesNotDestroysWhenNotExists(t *testing.T) {
	cc := config.NewContainer("tests")
	cc.Networks = []config.NetworkAttachment{config.NetworkAttachment{ Name: "cloud" }}
	md := &mocks.MockContainerTasks{}
	c := NewContainer(*cc, md, hclog.NewNullLogger())

	md.On("FindContainerIDs", cc.Name, cc.Type).Return(nil, nil)

	err := c.Destroy()
	assert.NoError(t, err)
	md.AssertNotCalled(t, "RemoveContainer")
}

func TestContainerDoesNotDestroysWhenLookupError(t *testing.T) {
	cc := config.NewContainer("tests")
	cc.Networks = []config.NetworkAttachment{config.NetworkAttachment{ Name: "cloud" }}
	md := &mocks.MockContainerTasks{}
	c := NewContainer(*cc, md, hclog.NewNullLogger())

	md.On("FindContainerIDs", cc.Name, cc.Type).Return(nil, fmt.Errorf("boom"))

	err := c.Destroy()
	assert.Error(t, err)
	md.AssertNotCalled(t, "RemoveContainer")
}

func TestContainerLooksupIDs(t *testing.T) {
	cc := config.NewContainer("tests")
	cc.Networks = []config.NetworkAttachment{config.NetworkAttachment{ Name: "cloud" }}
	md := &mocks.MockContainerTasks{}
	c := NewContainer(*cc, md, hclog.NewNullLogger())

	md.On("FindContainerIDs", cc.Name, cc.Type).Return([]string{"abc"}, nil)

	ids, err := c.Lookup()
	assert.NoError(t, err)
	assert.Equal(t, []string{"abc"}, ids)
}
