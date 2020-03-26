package clients

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/docker/pkg/ioutils"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupNomadClient() {

}

func createTestFile(t *testing.T) (string, string, *mocks.MockHTTP) {
	tmpDir, err := ioutils.TempDir("", "")
	assert.NoError(t, err)

	fp := filepath.Join(tmpDir, "nomad.json")
	f, err := os.Create(fp)
	assert.NoError(t, err)

	_, err = f.WriteString(getNomadConfig("localhost:4646"))
	assert.NoError(t, err)

	mh := &mocks.MockHTTP{}
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(validateResponse))),
		},
		nil,
	)

	return fp, tmpDir, mh
}

func TestNomadCreateReturnsErrorWhenFileNotExist(t *testing.T) {
	_, tmpDir, mh := createTestFile(t)
	defer os.RemoveAll(tmpDir)

	c := NewNomad(mh, hclog.NewNullLogger())
	err := c.Create([]string{"../../functional_tests/test_fixtures/nomad/example.nomad"}, false)
	assert.Error(t, err)
}

func TestNomadCreateValidatesConfig(t *testing.T) {
	fp, tmpDir, mh := createTestFile(t)
	defer os.RemoveAll(tmpDir)

	c := NewNomad(mh, hclog.NewNullLogger())
	c.SetConfig(fp)

	err := c.Create([]string{"../../functional_tests/test_fixtures/nomad/app_config/example.nomad"}, false)
	assert.NoError(t, err)

	mh.AssertCalled(t, "Do", mock.Anything)
}

func TestNomadCreateValidateErrorReturnsError(t *testing.T) {
	fp, tmpDir, mh := createTestFile(t)
	defer os.RemoveAll(tmpDir)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("Boom"))

	c := NewNomad(mh, hclog.NewNullLogger())
	c.SetConfig(fp)

	err := c.Create([]string{"../../functional_tests/test_fixtures/nomad/app_config/example.nomad"}, false)
	assert.Error(t, err)
}

func TestNomadCreateValidateNot200ReturnsError(t *testing.T) {
	fp, tmpDir, mh := createTestFile(t)
	defer os.RemoveAll(tmpDir)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(&http.Response{StatusCode: http.StatusInternalServerError}, nil)

	c := NewNomad(mh, hclog.NewNullLogger())
	c.SetConfig(fp)

	err := c.Create([]string{"../../functional_tests/test_fixtures/nomad/app_config/example.nomad"}, false)
	assert.Error(t, err)
}

func TestNomadCreateSubmitsJob(t *testing.T) {
	fp, tmpDir, mh := createTestFile(t)
	defer os.RemoveAll(tmpDir)

	c := NewNomad(mh, hclog.NewNullLogger())
	c.SetConfig(fp)

	err := c.Create([]string{"../../functional_tests/test_fixtures/nomad/app_config/example.nomad"}, false)
	assert.NoError(t, err)

	mh.AssertNumberOfCalls(t, "Do", 2)
}

func TestNomadCreateSubmitErrorReturnsError(t *testing.T) {
	fp, tmpDir, mh := createTestFile(t)
	defer os.RemoveAll(tmpDir)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte("testing"))),
		},
		nil,
	).Once()
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("Boom")).Once()

	c := NewNomad(mh, hclog.NewNullLogger())
	c.SetConfig(fp)

	err := c.Create([]string{"../../functional_tests/test_fixtures/nomad/app_config/example.nomad"}, false)
	assert.Error(t, err)
}

func TestNomadCreateSubmitNot200ReturnsError(t *testing.T) {
	fp, tmpDir, mh := createTestFile(t)
	defer os.RemoveAll(tmpDir)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte("testing"))),
		},
		nil,
	).Once()
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(&http.Response{StatusCode: http.StatusInternalServerError}, nil)

	c := NewNomad(mh, hclog.NewNullLogger())
	c.SetConfig(fp)

	err := c.Create([]string{"../../functional_tests/test_fixtures/nomad/app_config/example.nomad"}, false)
	assert.Error(t, err)
}

func TestNomadStopValidatesConfig(t *testing.T) {
	fp, tmpDir, mh := createTestFile(t)
	defer os.RemoveAll(tmpDir)

	c := NewNomad(mh, hclog.NewNullLogger())
	c.SetConfig(fp)

	err := c.Stop([]string{"../../functional_tests/test_fixtures/nomad/app_config/example.nomad"})
	assert.NoError(t, err)

	mh.AssertCalled(t, "Do", mock.Anything)
}

func TestNomadStopValidateErrorReturnsError(t *testing.T) {
	fp, tmpDir, mh := createTestFile(t)
	defer os.RemoveAll(tmpDir)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("Boom"))

	c := NewNomad(mh, hclog.NewNullLogger())
	c.SetConfig(fp)

	err := c.Stop([]string{"../../functional_tests/test_fixtures/nomad/app_config/example.nomad"})
	assert.Error(t, err)
}

func TestNomadStopStopsJob(t *testing.T) {
	fp, tmpDir, mh := createTestFile(t)
	defer os.RemoveAll(tmpDir)

	c := NewNomad(mh, hclog.NewNullLogger())
	c.SetConfig(fp)

	err := c.Stop([]string{"../../functional_tests/test_fixtures/nomad/app_config/example.nomad"})
	assert.NoError(t, err)

	mh.AssertNumberOfCalls(t, "Do", 2)
}

func TestNomadStopErrorReturnsError(t *testing.T) {
	fp, tmpDir, mh := createTestFile(t)
	defer os.RemoveAll(tmpDir)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte("testing"))),
		},
		nil,
	).Once()
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("boom"))

	c := NewNomad(mh, hclog.NewNullLogger())
	c.SetConfig(fp)

	err := c.Stop([]string{"../../functional_tests/test_fixtures/nomad/app_config/example.nomad"})
	assert.Error(t, err)
}

func TestNomadStopNoStatus200ReturnsError(t *testing.T) {
	fp, tmpDir, mh := createTestFile(t)
	defer os.RemoveAll(tmpDir)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte("testing"))),
		},
		nil,
	).Once()
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(&http.Response{StatusCode: http.StatusInternalServerError}, nil)

	c := NewNomad(mh, hclog.NewNullLogger())
	c.SetConfig(fp)

	err := c.Stop([]string{"../../functional_tests/test_fixtures/nomad/app_config/example.nomad"})
	assert.Error(t, err)
}
func TestNomadConfigLoadsCorrectly(t *testing.T) {
	fp, tmpDir, _ := createTestFile(t)
	defer os.RemoveAll(tmpDir)

	nc := &NomadConfig{}
	err := nc.Load(fp)
	assert.NoError(t, err)

	assert.Equal(t, "http://localhost:4646", nc.Location)
}

func TestNomadConfigLoadReturnsErrorWhenFileNotExist(t *testing.T) {
	nc := &NomadConfig{}
	err := nc.Load("file.json")
	assert.Error(t, err)
}

func TestNomadConfiSavesFile(t *testing.T) {
	fp, tmpDir, _ := createTestFile(t)
	defer os.RemoveAll(tmpDir)

	nc := &NomadConfig{Location: "http://nomad:4646"}
	err := nc.Save(fp)
	assert.NoError(t, err)

	// check the old file was deleted and the new file was written
	nc2 := &NomadConfig{}
	err = nc2.Load(fp)
	assert.NoError(t, err)

	assert.Equal(t, "http://nomad:4646", nc2.Location)
}

func getNomadConfig(l string) string {
	return fmt.Sprintf(`
	{
		"location": "http://%s"
	}`, l)
}

var validateResponse = `
{
  "AllAtOnce": false,
  "Constraints": null,
  "Affinities": null,
  "CreateIndex": 0,
  "Datacenters": null,
	"ID": "my-job"
}
`
