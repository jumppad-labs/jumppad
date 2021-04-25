package clients

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/go-hclog"
)

// HTTP defines an interface for a HTTP client
type HTTP interface {
	// HealthCheckHTTP makes a HTTP GET request to the given URI and
	// if a successful status []codes is returned the method returns a nil error.
	// If it is not possible to contact the URI or if any status other than the passed codes is returned
	// by the upstream, then the URI is retried until the timeout elapses.
	HealthCheckHTTP(uri string, codes []int, timeout time.Duration) error
	// Do executes a HTTP request and returns the response
	Do(r *http.Request) (*http.Response, error)
}

type HTTPImpl struct {
	backoff time.Duration
	httpc   *http.Client
	l       hclog.Logger
}

func NewHTTP(backoff time.Duration, l hclog.Logger) HTTP {
	httpc := &http.Client{}
	httpc.Transport = http.DefaultTransport.(*http.Transport).Clone()
	httpc.Transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	return &HTTPImpl{backoff, httpc, l}
}

// HealthCheckHTTP checks a http or HTTPS endpoint for a status 200
func (h *HTTPImpl) HealthCheckHTTP(address string, codes []int, timeout time.Duration) error {
	h.l.Debug("Performing health check for address", "address", address)
	st := time.Now()
	for {
		if time.Now().Sub(st) > timeout {
			h.l.Error("Timeout wating for HTTP healthcheck", "address", address)

			return fmt.Errorf("Timeout waiting for HTTP healthcheck %s", address)
		}

		resp, err := h.httpc.Get(address)
		if err == nil && assertResponseCode(codes, resp.StatusCode) {
			h.l.Debug("Health check complete", "address", address)
			return nil
		}

		// backoff
		time.Sleep(h.backoff)
	}
}

func assertResponseCode(codes []int, responseCode int) bool {
	for _, c := range codes {
		if responseCode == c {
			return true
		}
	}

	return false
}

// Do executes a HTTP request and returns the response
func (h *HTTPImpl) Do(r *http.Request) (*http.Response, error) {
	return h.httpc.Do(r)
}
