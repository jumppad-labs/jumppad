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
	"github.com/stretchr/testify/mock"
	assert "github.com/stretchr/testify/require"
)

func setupNomadClient() {

}

func setupNomadTests(t *testing.T) (string, string, *mocks.MockHTTP) {
	tmpDir, err := ioutils.TempDir("", "")
	assert.NoError(t, err)

	fp := filepath.Join(tmpDir, "nomad.json")
	f, err := os.Create(fp)
	assert.NoError(t, err)

	_, err = f.WriteString(getNomadConfig("localhost", 4646))
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
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(jobAllocationsResponse))),
		},
		nil,
	)

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp)

	s, err := c.JobRunning("test")
	assert.NoError(t, err)

	assert.True(t, s)
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

func TestNomadHealthWithNotReadyDockerRetries(t *testing.T) {
	fp, tmpDir, mh := setupNomadTests(t)
	defer os.RemoveAll(tmpDir)

	removeOn(&mh.Mock, "Do")

	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(unhealthyDockerResponse))),
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

func TestNomadEndpointsErrorWhenUnableToGetJobs(t *testing.T) {
	fp, tmpDir, mh := setupNomadTests(t)
	defer os.RemoveAll(tmpDir)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusBadRequest,
		},
		nil,
	)

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp)

	_, err := c.Endpoints("test", "test", "test")
	assert.Error(t, err)
}

func TestNomadEndpointsReturnsTwoEndpoints(t *testing.T) {
	t.Skip()
	fp, tmpDir, mh := setupNomadTests(t)
	defer os.RemoveAll(tmpDir)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(jobAllocationsResponse))),
		},
		nil,
	).Once()

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp)

	e, err := c.Endpoints("test", "test", "test")
	assert.NoError(t, err)
	assert.Len(t, e, 2)
}

func getNomadConfig(l string, p int) string {
	return fmt.Sprintf(`
	{
		"address": "%s",
		"api_port": %d,
		"node_count": 2
	}`, l, p)
}

var aliveResponse = `
[
	{
		"Name": "node1",
		"Status": "ready",
		"SchedulingEligibility": "eligible",
		"Drivers": {
			"docker": {
        "Healthy": true,
        "Detected": true
      }
		}
	},
	{
		"Name": "node2",
		"Status": "ready",
		"SchedulingEligibility": "eligible",
		"Drivers": {
			"docker": {
        "Healthy": true,
        "Detected": true
      }
		}
	}
]
`

var pendingResponse = `
[
	{
		"Name": "node1",
		"Status": "pending",
		"SchedulingEligibility": "eligible",
		"Drivers": {
			"docker": {
        "Healthy": true,
        "Detected": true
      }
		}
	},
	{
		"Name": "node2",
		"Status": "ready",
		"SchedulingEligibility": "eligible",
		"Drivers": {
			"docker": {
        "Healthy": true,
        "Detected": true
      }
		}
	}
]
`

var unhealthyDockerResponse = `
[
	{
		"Name": "node1",
		"Status": "ready",
		"SchedulingEligibility": "eligible",
		"Drivers": {
			"docker": {
        "Healthy": false
        "Detected": true
      }
		}
	},
	{
		"Name": "node2",
		"Status": "ready",
		"SchedulingEligibility": "eligible",
		"Drivers": {
			"docker": {
        "Healthy": true
        "Detected": true
      }
		}
	}
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
var jobAllocationsResponse = `
[
  {
    "ID": "da975cd1-8b04-6bce-9d5c-03e47353768c",
    "EvalID": "915e3cd4-81c6-dd1e-7880-55562ad938c6",
    "Name": "example_1.fake_service[0]",
    "Namespace": "default",
    "NodeID": "e92cfe74-1ba3-2248-cf89-18760af8c278",
    "NodeName": "server.dev",
    "JobID": "example_1",
    "JobType": "service",
    "JobVersion": 0,
    "TaskGroup": "fake_service",
    "DesiredStatus": "run",
    "DesiredDescription": "",
    "ClientStatus": "running"
  },
  {
    "ID": "da975cd1-8b04-6bce-9d5c-03e47353768c",
    "EvalID": "915e3cd4-81c6-dd1e-7880-55562ad938c6",
    "Name": "example_1.fake_service[0]",
    "Namespace": "default",
    "NodeID": "e92cfe74-1ba3-2248-cf89-18760af8c278",
    "NodeName": "server.dev",
    "JobID": "example_1",
    "JobType": "service",
    "JobVersion": 0,
    "TaskGroup": "fake_service",
    "DesiredStatus": "run",
    "DesiredDescription": "",
    "ClientStatus": "running"
  }
]
`

var allocationsResponse = `
{
  "ID": "da975cd1-8b04-6bce-9d5c-03e47353768c",
  "Namespace": "default",
  "EvalID": "915e3cd4-81c6-dd1e-7880-55562ad938c6",
  "Name": "example_1.fake_service[0]",
  "NodeID": "e92cfe74-1ba3-2248-cf89-18760af8c278",
  "NodeName": "server.dev",
  "JobID": "example_1",
  "Job": {
    "ID": "example_1",
    "Name": "example_1",
    "Datacenters": [
      "dc1"
    ],
    "Constraints": null,
    "Affinities": null,
    "Spreads": null,
    "TaskGroups": [
      {
        "Name": "fake_service",
        "Count": 1,
        "Tasks": [
          {
            "Name": "fake_service",
            "Driver": "docker",
            "Config": {
              "image": "nicholasjackson/fake-service:v0.18.1",
              "ports": [
                "http"
              ]
            }
          }
        ]
      }
    ]
  },
  "TaskGroup": "fake_service",
  "Resources": {
    "Networks": [
      {
        "IP": "10.5.0.2",
        "ReservedPorts": null,
        "DynamicPorts": [
          {
            "Label": "http",
            "Value": 28862,
            "To": 19090,
            "HostNetwork": "default"
          }
        ]
      }
    ]
  }
}
`
