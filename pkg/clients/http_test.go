package clients

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

func testSetupHTTPBasicServer(responseCode int) (string, *[]*http.Request, func()) {
	reqs := &[]*http.Request{}
	s := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		fmt.Println("testing")
		*reqs = append(*reqs, r)
		rw.WriteHeader(responseCode)
	}))

	return s.URL, reqs, func() {
		s.Close()
	}
}

func TestHTTPCallsGet(t *testing.T) {
	url, reqs, cleanup := testSetupHTTPBasicServer(http.StatusOK)
	defer cleanup()

	c := NewHTTP(1*time.Millisecond, hclog.NewNullLogger())

	err := c.HealthCheckHTTP(url, 10*time.Millisecond)
	assert.NoError(t, err)
	assert.Len(t, *reqs, 1)
}

func TestHTTPRetryiesOnServerErrorCode(t *testing.T) {
	url, reqs, cleanup := testSetupHTTPBasicServer(http.StatusBadRequest)
	defer cleanup()

	c := NewHTTP(1*time.Millisecond, hclog.NewNullLogger())

	err := c.HealthCheckHTTP(url, 10*time.Millisecond)
	assert.Error(t, err)
	assert.Greater(t, len(*reqs), 1)
}

func TestHTTPErrorsOnClientError(t *testing.T) {
	_, reqs, cleanup := testSetupHTTPBasicServer(http.StatusBadRequest)
	defer cleanup()

	c := NewHTTP(1*time.Millisecond, hclog.NewNullLogger())

	err := c.HealthCheckHTTP("http://127.0.0.2:9090", 10*time.Millisecond)
	assert.Error(t, err)
	assert.Len(t, *reqs, 0)
}
