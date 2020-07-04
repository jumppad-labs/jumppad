package clients

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/pkg/ioutils"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupNomadClient() {

}

func setupNomadTests(t *testing.T) (string, string, *mocks.MockHTTP) {
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
	_, tmpDir, mh := setupNomadTests(t)
	defer os.RemoveAll(tmpDir)

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	err := c.Create([]string{"../../examples/nomad/example.nomad"})
	assert.Error(t, err)
}

func TestNomadCreateValidatesConfig(t *testing.T) {
	fp, tmpDir, mh := setupNomadTests(t)
	defer os.RemoveAll(tmpDir)

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp)

	err := c.Create([]string{"../../examples/nomad/app_config/example.nomad"})
	assert.NoError(t, err)

	mh.AssertCalled(t, "Do", mock.Anything)
}

func TestNomadCreateValidateErrorReturnsError(t *testing.T) {
	fp, tmpDir, mh := setupNomadTests(t)
	defer os.RemoveAll(tmpDir)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("Boom"))

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp)

	err := c.Create([]string{"../../examples/nomad/app_config/example.nomad"})
	assert.Error(t, err)
}

func TestNomadCreateValidateNot200ReturnsError(t *testing.T) {
	fp, tmpDir, mh := setupNomadTests(t)
	defer os.RemoveAll(tmpDir)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(&http.Response{StatusCode: http.StatusInternalServerError}, nil)

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp)

	err := c.Create([]string{"../../examples/nomad/app_config/example.nomad"})
	assert.Error(t, err)
}

func TestNomadCreateSubmitsJob(t *testing.T) {
	fp, tmpDir, mh := setupNomadTests(t)
	defer os.RemoveAll(tmpDir)

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp)

	err := c.Create([]string{"../../examples/nomad/app_config/example.nomad"})
	assert.NoError(t, err)

	mh.AssertNumberOfCalls(t, "Do", 2)
}

func TestNomadCreateSubmitErrorReturnsError(t *testing.T) {
	fp, tmpDir, mh := setupNomadTests(t)
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

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp)

	err := c.Create([]string{"../../examples/nomad/app_config/example.nomad"})
	assert.Error(t, err)
}

func TestNomadCreateSubmitNot200ReturnsError(t *testing.T) {
	fp, tmpDir, mh := setupNomadTests(t)
	defer os.RemoveAll(tmpDir)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte("testing"))),
		},
		nil,
	).Once()

	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte("testing"))),
		},
		nil,
	)

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp)

	err := c.Create([]string{"../../examples/nomad/app_config/example.nomad"})
	assert.Error(t, err)
}

func TestNomadStopValidatesConfig(t *testing.T) {
	fp, tmpDir, mh := setupNomadTests(t)
	defer os.RemoveAll(tmpDir)

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp)

	err := c.Stop([]string{"../../examples/nomad/app_config/example.nomad"})
	assert.NoError(t, err)

	mh.AssertCalled(t, "Do", mock.Anything)
}

func TestNomadStopValidateErrorReturnsError(t *testing.T) {
	fp, tmpDir, mh := setupNomadTests(t)
	defer os.RemoveAll(tmpDir)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("Boom"))

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp)

	err := c.Stop([]string{"../../examples/nomad/app_config/example.nomad"})
	assert.Error(t, err)
}

func TestNomadStopStopsJob(t *testing.T) {
	fp, tmpDir, mh := setupNomadTests(t)
	defer os.RemoveAll(tmpDir)

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp)

	err := c.Stop([]string{"../../examples/nomad/app_config/example.nomad"})
	assert.NoError(t, err)

	mh.AssertNumberOfCalls(t, "Do", 2)
}

func TestNomadStopErrorReturnsError(t *testing.T) {
	fp, tmpDir, mh := setupNomadTests(t)
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

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp)

	err := c.Stop([]string{"../../examples/nomad/app_config/example.nomad"})
	assert.Error(t, err)
}

func TestNomadStopNoStatus200ReturnsError(t *testing.T) {
	fp, tmpDir, mh := setupNomadTests(t)
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

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp)

	err := c.Stop([]string{"../../examples/nomad/app_config/example.nomad"})
	assert.Error(t, err)
}

func TestNomadJobStatusReturnsStatus(t *testing.T) {
	fp, tmpDir, mh := setupNomadTests(t)
	defer os.RemoveAll(tmpDir)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(allocationsResponse))),
		},
		nil,
	)

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp)

	s, err := c.JobStatus("test")
	assert.NoError(t, err)

	assert.Equal(t, "running", s)
}

func TestNomadHealthCallsAPI(t *testing.T) {
	fp, tmpDir, mh := setupNomadTests(t)
	defer os.RemoveAll(tmpDir)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(aliveResponse))),
		},
		nil,
	).Once()

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp)

	err := c.HealthCheckAPI(10 * time.Millisecond)
	assert.NoError(t, err)
}

func TestNomadHealthWithNotReadyNodeRetries(t *testing.T) {
	fp, tmpDir, mh := setupNomadTests(t)
	defer os.RemoveAll(tmpDir)

	removeOn(&mh.Mock, "Do")

	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(pendingResponse))),
		},
		nil,
	).Once()

	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(aliveResponse))),
		},
		nil,
	).Once()

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp)

	err := c.HealthCheckAPI(10 * time.Millisecond)
	assert.NoError(t, err)
	mh.AssertNumberOfCalls(t, "Do", 2)
}

func TestNomadHealthErrorsOnClientError(t *testing.T) {
	fp, tmpDir, mh := setupNomadTests(t)
	defer os.RemoveAll(tmpDir)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		nil,
		fmt.Errorf("boom"),
	)

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp)

	err := c.HealthCheckAPI(10 * time.Millisecond)
	assert.Error(t, err)
}

func getNomadConfig(l string) string {
	return fmt.Sprintf(`
	{
		"location": "http://%s",
		"node_count": 2
	}`, l)
}

var aliveResponse = `
[
	{"Status": "ready"},
	{"Status": "ready"}
]
`

var pendingResponse = `
[
	{"Status": "pending"},
	{"Status": "ready"}
]
`

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
var allocationsResponse = `
{
    "ID": "ed344e0a-7290-d117-41d3-a64f853ca3c2",
		"JobID": "example",
		"Status": "running",
    "TaskGroup": "cache",
    "TaskStates": {
      "redis": {
				"State": "running"
			},
      "web": {
				"State": "running"
			}
		}
	},
}
`
