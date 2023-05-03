package clients

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/jumppad/pkg/clients/mocks"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/stretchr/testify/mock"
	assert "github.com/stretchr/testify/require"
)

func setupNomadClient() {

}

func setupNomadTests(t *testing.T) (utils.ClusterConfig, string, *mocks.MockHTTP) {
	tmpDir := t.TempDir()

	home := os.Getenv(utils.HomeEnvName())
	os.Setenv(utils.HomeEnvName(), tmpDir)
	t.Cleanup(func() {
		os.Setenv(utils.HomeEnvName(), home)
	})

	mh := &mocks.MockHTTP{}
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(validateResponse))),
		},
		nil,
	)

	clusterConfig, _ := utils.GetClusterConfig("nomad_cluster." + "testing")
	clusterConfig.NodeCount = 2

	return clusterConfig, tmpDir, mh
}

func TestNomadCreateReturnsErrorWhenFileNotExist(t *testing.T) {
	_, _, mh := setupNomadTests(t)

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	err := c.Create([]string{"../../examples/nomad/example.nomad"})
	assert.Error(t, err)
}

func TestNomadCreateValidatesConfig(t *testing.T) {
	fp, _, mh := setupNomadTests(t)

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp, "local")

	err := c.Create([]string{"../../examples/nomad/app_config/example.nomad"})
	assert.NoError(t, err)

	mh.AssertCalled(t, "Do", mock.Anything)
}

func TestNomadCreateValidateErrorReturnsError(t *testing.T) {
	fp, _, mh := setupNomadTests(t)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("Boom"))

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp, "local")

	err := c.Create([]string{"../../examples/nomad/app_config/example.nomad"})
	assert.Error(t, err)
}

func TestNomadCreateValidateNot200ReturnsError(t *testing.T) {
	fp, _, mh := setupNomadTests(t)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(&http.Response{StatusCode: http.StatusInternalServerError}, nil)

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp, "local")

	err := c.Create([]string{"../../examples/nomad/app_config/example.nomad"})
	assert.Error(t, err)
}

func TestNomadCreateValidateInvalidReturnsError(t *testing.T) {
	fp, _, mh := setupNomadTests(t)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       ioutil.NopCloser(bytes.NewBufferString("oops")),
		}, nil)

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp, "local")

	err := c.Create([]string{"../../examples/nomad/app_config/example.nomad"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "oops")
}

func TestNomadCreateSubmitsJob(t *testing.T) {
	fp, _, mh := setupNomadTests(t)

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp, "local")

	err := c.Create([]string{"../../examples/nomad/app_config/example.nomad"})
	assert.NoError(t, err)

	mh.AssertNumberOfCalls(t, "Do", 2)
}

func TestNomadCreateSubmitErrorReturnsError(t *testing.T) {
	fp, _, mh := setupNomadTests(t)

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
	c.SetConfig(fp, "local")

	err := c.Create([]string{"../../examples/nomad/app_config/example.nomad"})
	assert.Error(t, err)
}

func TestNomadCreateSubmitNot200ReturnsError(t *testing.T) {
	fp, _, mh := setupNomadTests(t)

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
	c.SetConfig(fp, "local")

	err := c.Create([]string{"../../examples/nomad/app_config/example.nomad"})
	assert.Error(t, err)
}

func TestNomadStopValidatesConfig(t *testing.T) {
	fp, _, mh := setupNomadTests(t)

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp, "local")

	err := c.Stop([]string{"../../examples/nomad/app_config/example.nomad"})
	assert.NoError(t, err)

	mh.AssertCalled(t, "Do", mock.Anything)
}

func TestNomadStopValidateErrorReturnsError(t *testing.T) {
	fp, _, mh := setupNomadTests(t)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("Boom"))

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp, "local")

	err := c.Stop([]string{"../../examples/nomad/app_config/example.nomad"})
	assert.Error(t, err)
}

func TestNomadStopStopsJob(t *testing.T) {
	fp, _, mh := setupNomadTests(t)

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp, "local")

	err := c.Stop([]string{"../../examples/nomad/app_config/example.nomad"})
	assert.NoError(t, err)

	mh.AssertNumberOfCalls(t, "Do", 2)
}

func TestNomadStopErrorReturnsError(t *testing.T) {
	fp, _, mh := setupNomadTests(t)

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
	c.SetConfig(fp, "local")

	err := c.Stop([]string{"../../examples/nomad/app_config/example.nomad"})
	assert.Error(t, err)
}

func TestNomadStopNoStatus200ReturnsError(t *testing.T) {
	fp, _, mh := setupNomadTests(t)

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
	c.SetConfig(fp, "local")

	err := c.Stop([]string{"../../examples/nomad/app_config/example.nomad"})
	assert.Error(t, err)
}

func TestNomadJobStatusReturnsNoErrorOnRunning(t *testing.T) {
	fp, _, mh := setupNomadTests(t)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(jobAllocationsResponse))),
		},
		nil,
	)

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp, "local")

	s, err := c.JobRunning("test")
	assert.NoError(t, err)

	assert.True(t, s)
}

func TestNomadJobStatusReturnsErrorWhenPending(t *testing.T) {
	fp, _, mh := setupNomadTests(t)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(jobAllocationsPendingResponse))),
		},
		nil,
	)

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp, "local")

	s, err := c.JobRunning("test")
	assert.NoError(t, err)

	assert.False(t, s)
}

func TestNomadHealthCallsAPI(t *testing.T) {
	fp, _, mh := setupNomadTests(t)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(aliveResponse))),
		},
		nil,
	).Once()

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp, "local")

	err := c.HealthCheckAPI(10 * time.Millisecond)
	assert.NoError(t, err)
}

func TestNomadHealthWithNotReadyNodeRetries(t *testing.T) {
	fp, _, mh := setupNomadTests(t)

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
	c.SetConfig(fp, "local")

	err := c.HealthCheckAPI(10 * time.Millisecond)
	assert.NoError(t, err)
	mh.AssertNumberOfCalls(t, "Do", 2)
}

func TestNomadHealthWithNotReadyDockerRetries(t *testing.T) {
	fp, _, mh := setupNomadTests(t)

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
	c.SetConfig(fp, "local")

	err := c.HealthCheckAPI(10 * time.Millisecond)
	assert.NoError(t, err)
	mh.AssertNumberOfCalls(t, "Do", 2)

}

func TestNomadHealthErrorsOnClientError(t *testing.T) {
	fp, _, mh := setupNomadTests(t)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		nil,
		fmt.Errorf("boom"),
	)

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp, "local")

	err := c.HealthCheckAPI(10 * time.Millisecond)
	assert.Error(t, err)
}

func TestNomadEndpointsErrorWhenUnableToGetJobs(t *testing.T) {
	fp, _, mh := setupNomadTests(t)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusBadRequest,
		},
		nil,
	)

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp, "local")

	_, err := c.Endpoints("test", "test", "test")
	assert.Error(t, err)
}

func TestNomadEndpointsReturnsTwoEndpoints(t *testing.T) {
	fp, _, mh := setupNomadTests(t)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(jobAllocationsResponse2Running))),
		},
		nil,
	).Once()

	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(allocationsResponse1))),
		},
		nil,
	).Once()

	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(allocationsResponse2))),
		},
		nil,
	).Once()

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp, "local")

	e, err := c.Endpoints("example_1", "fake_service", "fake_service")
	assert.NoError(t, err)
	assert.Len(t, e, 2)

	assert.Equal(t, "10.5.0.2:28862", e[0]["http"])
	assert.Equal(t, "10.5.0.3:19090", e[1]["http"])
}

func TestNomadEndpointsReturnsRunningEndpoints(t *testing.T) {
	fp, _, mh := setupNomadTests(t)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(jobAllocationsResponse))),
		},
		nil,
	).Once()

	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(allocationsResponse1))),
		},
		nil,
	).Once()

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp, "local")

	e, err := c.Endpoints("example_1", "fake_service", "fake_service")
	assert.NoError(t, err)
	assert.Len(t, e, 1)

	assert.Equal(t, "10.5.0.2:28862", e[0]["http"])
}
func TestNomadEndpointsReturnsConnectEndpoints(t *testing.T) {
	fp, _, mh := setupNomadTests(t)

	removeOn(&mh.Mock, "Do")
	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(jobAllocationsConnectResponse))),
		},
		nil,
	).Once()

	mh.On("Do", mock.Anything, mock.Anything, mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(allocationsConnectResponse))),
		},
		nil,
	).Once()

	c := NewNomad(mh, 1*time.Millisecond, hclog.NewNullLogger())
	c.SetConfig(fp, "local")

	e, err := c.Endpoints("web", "web", "web")
	assert.NoError(t, err)
	assert.Len(t, e, 1)

	assert.Equal(t, "10.5.0.4:9090", e[0]["http"])
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
    "ClientStatus": "complete"
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

var jobAllocationsResponse2Running = `
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

var jobAllocationsPendingResponse = `
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
    "ClientStatus": "pending"
  }
]
`

var allocationsResponse1 = `
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

var allocationsResponse2 = `
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
        "IP": "10.5.0.3",
        "ReservedPorts": [
          {
            "Label": "http",
            "Value": 19090,
            "To": 19090,
            "HostNetwork": "default"
          }
        ]
      }
    ]
  }
}
`

var jobAllocationsConnectResponse = `
[
  {
    "ID": "da975cd1-8b04-6bce-9d5c-03e47353768c",
  	"EvalID": "02ee5334-be8f-fcba-41a0-cbbae8e34fce",
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
  	"EvalID": "02ee5334-be8f-fcba-41a0-cbbae8e34fce",
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
    "ClientStatus": "complete"
  }
]
`
var allocationsConnectResponse = `
{
  "ID": "c64cec54-843a-b660-c8ff-386bcaaaef88",
  "Namespace": "default",
  "EvalID": "02ee5334-be8f-fcba-41a0-cbbae8e34fce",
  "Name": "web.web[0]",
  "NodeID": "fd7e30b8-80e6-5cb0-980e-01913a8c6666",
  "NodeName": "server.local",
  "JobID": "web",
  "Job": {
    "Stop": false,
    "Region": "global",
    "Namespace": "default",
    "ID": "web",
    "ParentID": "",
    "Name": "web",
    "Type": "service",
    "Priority": 50,
    "AllAtOnce": false,
    "Datacenters": [
      "dc1"
    ],
    "Constraints": null,
    "Affinities": null,
    "Spreads": null,
    "TaskGroups": [
      {
        "Name": "web",
        "Count": 1,
        "Tasks": [
          {
            "Name": "web",
            "Driver": "docker",
            "User": "",
            "Config": {
              "image": "nicholasjackson/fake-service:v0.9.0"
            }
          },
          {
            "Name": "connect-proxy-web",
            "Driver": "docker",
            "User": ""
          }
        ],
        "Networks": [
          {
            "Mode": "bridge",
            "Device": "",
            "CIDR": "",
            "IP": "",
            "MBits": 10,
            "ReservedPorts": [
              {
                "Label": "http",
                "Value": 9090,
                "To": 9090
              }
            ],
            "DynamicPorts": [
              {
                "Label": "connect-proxy-web",
                "Value": 0,
                "To": -1
              }
            ]
          }
        ],
        "Services": [
          {
            "Name": "web",
            "PortLabel": "9090",
            "AddressMode": "auto",
            "EnableTagOverride": false,
            "Tags": [
              "global",
              "app"
            ],
            "CanaryTags": null,
            "Checks": null,
            "Connect": {
            },
            "Meta": null,
            "CanaryMeta": null
          }
        ],
        "Volumes": null,
        "ShutdownDelay": null
      }
    ]
  },
  "TaskGroup": "web",
  "Resources": {
    "CPU": 750,
    "MemoryMB": 384,
    "Networks": [
      {
        "Mode": "bridge",
        "Device": "eth0",
        "IP": "10.5.0.4",
        "ReservedPorts": [
          {
            "Label": "http",
            "Value": 9090,
            "To": 9090
          }
        ],
        "DynamicPorts": [
          {
            "Label": "connect-proxy-web",
            "Value": 27144,
            "To": 27144
          }
        ]
      }
    ],
    "Devices": null
  },
  "SharedResources": {
    "CPU": 0,
    "MemoryMB": 0,
    "DiskMB": 30,
    "IOPS": 0,
    "Networks": [
      {
        "Mode": "bridge",
        "Device": "eth0",
        "CIDR": "",
        "IP": "10.5.0.4",
        "MBits": 10,
        "ReservedPorts": [
          {
            "Label": "http",
            "Value": 9090,
            "To": 9090
          }
        ],
        "DynamicPorts": [
          {
            "Label": "connect-proxy-web",
            "Value": 27144,
            "To": 27144
          }
        ]
      }
    ],
    "Devices": null
  },
  "AllocatedResources": {
    "Tasks": {
      "web": {
        "Cpu": {
          "CpuShares": 500
        },
        "Memory": {
          "MemoryMB": 256
        },
        "Networks": null,
        "Devices": null
      },
      "connect-proxy-web": {
        "Cpu": {
          "CpuShares": 250
        },
        "Memory": {
          "MemoryMB": 128
        },
        "Networks": null,
        "Devices": null
      }
    },
    "TaskLifecycles": {
      "web": null,
      "connect-proxy-web": {
        "Hook": "prestart",
        "Sidecar": true
      }
    },
    "Shared": {
      "Networks": [
        {
          "Mode": "bridge",
          "Device": "eth0",
          "CIDR": "",
          "IP": "10.5.0.4",
          "MBits": 10,
          "ReservedPorts": [
            {
              "Label": "http",
              "Value": 9090,
              "To": 9090
            }
          ],
          "DynamicPorts": [
            {
              "Label": "connect-proxy-web",
              "Value": 27144,
              "To": 27144
            }
          ]
        }
      ],
      "DiskMB": 30
    }
  },
  "DesiredStatus": "run",
  "ClientStatus": "running",
  "ClientDescription": "Tasks are running",
  "DeploymentID": "8154fd6a-830c-117d-3a48-b9c1454507c0",
  "DeploymentStatus": {
    "Healthy": true,
    "Timestamp": "2021-03-22T07:20:45.5071198Z",
    "Canary": false,
    "ModifyIndex": 42
  },
  "CreateIndex": 15,
  "ModifyIndex": 42,
  "AllocModifyIndex": 15,
  "CreateTime": 1616397620428715000,
  "ModifyTime": 1616397645647263000
}
`
