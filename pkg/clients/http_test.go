package clients

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

func testSetupHTTPBasicServer(responseCode int, body string) (string, *[]*http.Request, func()) {
	reqs := &[]*http.Request{}
	s := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		*reqs = append(*reqs, r)
		rw.WriteHeader(responseCode)
		rw.Write([]byte(body))
	}))

	return s.URL, reqs, func() {
		s.Close()
	}
}

func TestHTTPHealthCallsGet(t *testing.T) {
	url, reqs, cleanup := testSetupHTTPBasicServer(http.StatusOK, "")
	defer cleanup()

	c := NewHTTP(1*time.Millisecond, hclog.NewNullLogger())

	err := c.HealthCheckHTTP(url, 10*time.Millisecond)
	assert.NoError(t, err)
	assert.Len(t, *reqs, 1)
}

func TestHTTPHealthRetryiesOnServerErrorCode(t *testing.T) {
	url, reqs, cleanup := testSetupHTTPBasicServer(http.StatusBadRequest, "")
	defer cleanup()

	c := NewHTTP(1*time.Millisecond, hclog.NewNullLogger())

	err := c.HealthCheckHTTP(url, 10*time.Millisecond)
	assert.Error(t, err)
	assert.Greater(t, len(*reqs), 1)
}

func TestHTTPHealthErrorsOnClientError(t *testing.T) {
	_, reqs, cleanup := testSetupHTTPBasicServer(http.StatusBadRequest, "")
	defer cleanup()

	c := NewHTTP(1*time.Millisecond, hclog.NewNullLogger())

	err := c.HealthCheckHTTP("http://127.0.0.2:9090", 10*time.Millisecond)
	assert.Error(t, err)
	assert.Len(t, *reqs, 0)
}

func TestHTTPNomadCallsAPI(t *testing.T) {
	url, reqs, cleanup := testSetupHTTPBasicServer(http.StatusOK,
		`
		[
			{"Status": "ready"},
			{"Status": "ready"}
		]
		`,
	)
	defer cleanup()

	c := NewHTTP(1*time.Millisecond, hclog.NewNullLogger())

	err := c.HealthCheckNomad(url, 2, 10*time.Millisecond)
	assert.NoError(t, err)
	assert.Len(t, *reqs, 1)
}

func TestHTTPNomadWithNotReadyNodeRetries(t *testing.T) {
	url, reqs, cleanup := testSetupHTTPBasicServer(http.StatusOK,
		`
		[
			{"Status": "pending"},
			{"Status": "ready"}
		]
		`,
	)
	defer cleanup()

	c := NewHTTP(1*time.Millisecond, hclog.NewNullLogger())

	err := c.HealthCheckNomad(url, 2, 10*time.Millisecond)
	assert.Error(t, err)
	assert.Greater(t, len(*reqs), 1)
}

func TestHTTPNomadErrorsOnClientError(t *testing.T) {
	_, reqs, cleanup := testSetupHTTPBasicServer(http.StatusBadRequest, "")
	defer cleanup()

	c := NewHTTP(1*time.Millisecond, hclog.NewNullLogger())

	err := c.HealthCheckNomad("http://127.0.0.2:9090", 2, 10*time.Millisecond)
	assert.Error(t, err)
	assert.Len(t, *reqs, 0)
}
