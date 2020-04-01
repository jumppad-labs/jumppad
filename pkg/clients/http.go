package clients

import (
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/go-hclog"
)

// HTTP defines an interface for a HTTP client
type HTTP interface {
	// HealthCheckHTTP makes a HTTP GET request to the given URI and
	// if a successful status 200 is returned the method returns a nil error.
	// If it is not possible to contact the URI or if any status other than 200 is returned
	// by the upstream, then the URI is retried until the timeout elapses.
	HealthCheckHTTP(uri string, timeout time.Duration) error
	// Do executes a HTTP request and returns the response
	Do(r *http.Request) (*http.Response, error)
}

type HTTPImpl struct {
	backoff time.Duration
	l       hclog.Logger
}

func NewHTTP(d time.Duration, l hclog.Logger) HTTP {
	return &HTTPImpl{d, l}
}

func (h *HTTPImpl) HealthCheckHTTP(address string, timeout time.Duration) error {
	h.l.Debug("Performing health check for address", "address", address)
	st := time.Now()
	for {
		if time.Now().Sub(st) > timeout {
			h.l.Error("Timeout wating for HTTP healthcheck", "address", address)

			return fmt.Errorf("Timeout waiting for HTTP healthcheck %s", address)
		}

		resp, err := http.Get(address)
		if err == nil && resp.StatusCode == 200 {
			h.l.Debug("Health check complete", "address", address)
			return nil
		}

		// backoff
		time.Sleep(h.backoff)
	}
}

// Do executes a HTTP request and returns the response
func (h *HTTPImpl) Do(r *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(r)
}
