package providers

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupDocs() (*Docs, *mocks.MockContainerTasks) {
	cc := config.NewDocs("tests")
	cc.IndexTitle = "test"
	cc.IndexPages = []string{"abc", "123"}
	cc.Path = "./docs"

	md := &mocks.MockContainerTasks{}

	md.On("PullImage", mock.Anything, false).Return(nil)
	md.On("CreateContainer", mock.Anything).Return("", nil)
	md.On("FindContainerIDs", mock.Anything, mock.Anything).Return([]string{"abc"}, nil)
	md.On("RemoveContainer", mock.Anything).Return(nil)

	d := NewDocs(cc, md, hclog.NewNullLogger())

	return d, md
}

func TestDocsPullsDocsContainer(t *testing.T) {
	d, md := setupDocs()

	err := d.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "PullImage")[0].Arguments[0].(config.Image)
	assert.Equal(t, params.Name, docsImageName+":"+docsVersion)
}

func TestDocsMountsMarkdown(t *testing.T) {
	d, md := setupDocs()

	err := d.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*config.Container)

	// check the config file has been generated
	// this will be the second volume
	assert.Equal(t, d.config.Path, params.Volumes[0].Source)
	assert.Equal(t, "/shipyard/docs", params.Volumes[0].Destination)
}

func TestDocsGeneratesDocusaurusConfig(t *testing.T) {
	d, md := setupDocs()

	err := d.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*config.Container)

	// check the config file has been generated
	// this will be the second volume
	fp := params.Volumes[1].Source
	assert.FileExists(t, fp)

	// check the file is mounted correctly
	assert.Equal(t, "/shipyard/sidebars.js", params.Volumes[1].Destination)

	//check the file has been generated correctlly
	data, err := ioutil.ReadFile(fp)
	assert.Contains(t, string(data), `"abc",`)
	assert.Contains(t, string(data), `"123"`)
}

func TestDocsSetsDocsPorts(t *testing.T) {
	d, md := setupDocs()

	err := d.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*config.Container)

	// main port
	assert.Equal(t, "80", params.Ports[0].Local)
	assert.Equal(t, "80", params.Ports[0].Remote)
	assert.Equal(t, fmt.Sprintf("%d", d.config.Port), params.Ports[0].Host)

	// livereload
	assert.Equal(t, "37950", params.Ports[1].Local)
	assert.Equal(t, "37950", params.Ports[1].Remote)
	assert.Equal(t, "37950", params.Ports[1].Host)
}

func TestDocsSetsDocsPortsWithCustomReload(t *testing.T) {
	d, md := setupDocs()
	d.config.LiveReloadPort = 30000

	err := d.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[0].Arguments[0].(*config.Container)

	// main port
	assert.Equal(t, "80", params.Ports[0].Local)
	assert.Equal(t, "80", params.Ports[0].Remote)
	assert.Equal(t, fmt.Sprintf("%d", d.config.Port), params.Ports[0].Host)

	// livereload
	assert.Equal(t, "37950", params.Ports[1].Local)
	assert.Equal(t, "37950", params.Ports[1].Remote)
	assert.Equal(t, "30000", params.Ports[1].Host)
}

func TestDocsPullsTerminalContainer(t *testing.T) {
	d, md := setupDocs()

	err := d.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "PullImage")[1].Arguments[0].(config.Image)
	assert.Equal(t, params.Name, terminalImageName+":"+terminalVersion)
}

func TestDocsMountsDockerSock(t *testing.T) {
	d, md := setupDocs()

	err := d.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[1].Arguments[0].(*config.Container)

	// check the config file has been generated
	// this will be the second volume
	assert.Equal(t, utils.GetDockerSock(), params.Volumes[0].Source)
	assert.Equal(t, utils.GetDockerSock(), params.Volumes[0].Destination)
}

func TestDocsSetsTerminalPorts(t *testing.T) {
	d, md := setupDocs()

	err := d.Create()
	assert.NoError(t, err)

	params := getCalls(&md.Mock, "CreateContainer")[1].Arguments[0].(*config.Container)

	// main port
	assert.Equal(t, "27950", params.Ports[0].Host)
	assert.Equal(t, "27950", params.Ports[0].Local)
}

func TestDestroyRemovesContainers(t *testing.T) {
	d, md := setupDocs()

	err := d.Create()
	assert.NoError(t, err)

	err = d.Destroy()
	assert.NoError(t, err)

	md.AssertNumberOfCalls(t, "FindContainerIDs", 2)
	md.AssertNumberOfCalls(t, "RemoveContainer", 2)
}
