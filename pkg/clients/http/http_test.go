package http

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
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

	c := NewHTTP(1*time.Millisecond, logger.NewTestLogger(t))

	err := c.HealthCheckHTTP(url, "", nil, "", []int{200}, 10*time.Millisecond)
	assert.NoError(t, err)
	assert.Len(t, *reqs, 1)
}

func TestHTTPHealthCallsGetMultipleStatusCodes(t *testing.T) {
	url, reqs, cleanup := testSetupHTTPBasicServer(http.StatusNoContent, "")
	defer cleanup()

	c := NewHTTP(1*time.Millisecond, logger.NewTestLogger(t))

	err := c.HealthCheckHTTP(url, "", nil, "", []int{200, 204}, 10*time.Millisecond)
	assert.NoError(t, err)
	assert.Len(t, *reqs, 1)
}

func TestHTTPHealthRetryiesOnServerErrorCode(t *testing.T) {
	url, reqs, cleanup := testSetupHTTPBasicServer(http.StatusBadRequest, "")
	defer cleanup()

	c := NewHTTP(1*time.Millisecond, logger.NewTestLogger(t))

	err := c.HealthCheckHTTP(url, "", nil, "", []int{200}, 50*time.Millisecond)
	assert.Error(t, err)
	assert.Greater(t, len(*reqs), 1)
}

func TestHTTPHealthErrorsOnClientError(t *testing.T) {
	_, reqs, cleanup := testSetupHTTPBasicServer(http.StatusBadRequest, "")
	defer cleanup()

	c := NewHTTP(1*time.Millisecond, logger.NewTestLogger(t), WithTransport(&http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Millisecond,
			KeepAlive: 10 * time.Millisecond,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Millisecond,
	}))

	err := c.HealthCheckHTTP("http://127.0.0.2:19091", "", nil, "", []int{200}, 10*time.Millisecond)
	assert.Error(t, err)
	assert.Len(t, *reqs, 0)
}

func TestHTTPHealthSetsHostOnHostHeader(t *testing.T) {
	url, reqs, cleanup := testSetupHTTPBasicServer(http.StatusOK, "")
	defer cleanup()

	c := NewHTTP(1*time.Millisecond, logger.NewTestLogger(t))

	err := c.HealthCheckHTTP(url, "", map[string][]string{"Host": {"example.com"}}, "", []int{200}, 10*time.Millisecond)
	assert.NoError(t, err)
	assert.Len(t, *reqs, 1)
	assert.Equal(t, "example.com", (*reqs)[0].Host)
}

func TestHTTPHealthSetsHostOnHostHeaderWithMultipleValues(t *testing.T) {
	url, reqs, cleanup := testSetupHTTPBasicServer(http.StatusOK, "")
	defer cleanup()

	c := NewHTTP(1*time.Millisecond, logger.NewTestLogger(t))

	err := c.HealthCheckHTTP(url, "", map[string][]string{"Host": {"example.com", "example.org"}}, "", []int{200}, 10*time.Millisecond)
	assert.NoError(t, err)
	assert.Len(t, *reqs, 1)
	assert.Equal(t, "example.com", (*reqs)[0].Host)
}

func TestHTTPHealthSetsHostOnHostHeaderWithNoValues(t *testing.T) {
	url, reqs, cleanup := testSetupHTTPBasicServer(http.StatusOK, "")
	defer cleanup()

	host := strings.TrimPrefix(url, "http://")

	c := NewHTTP(1*time.Millisecond, logger.NewTestLogger(t))

	err := c.HealthCheckHTTP(url, "", map[string][]string{}, "", []int{200}, 10*time.Millisecond)
	assert.NoError(t, err)
	assert.Len(t, *reqs, 1)
	assert.Equal(t, host, (*reqs)[0].Host)
}
