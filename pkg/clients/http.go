package clients

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net"
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

	HealthCheckHTTP(uri, method string, headers map[string][]string, body string, codes []int, timeout time.Duration) error

	// HealthCheckTCP attempts to connect to a raw socket at the given address
	// if a connection is established the health check is marked as a success
	// if failed the check will retry until the timeout occurs
	HealthCheckTCP(uri string, timeout time.Duration) error
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
func (h *HTTPImpl) HealthCheckHTTP(address, method string, headers map[string][]string, body string, codes []int, timeout time.Duration) error {
	h.l.Debug("Performing HTTP health check for address", "address", address, "method", method, "headers", headers, "body", body, "codes", codes)
	st := time.Now()
	for {
		if time.Since(st) > timeout {
			h.l.Error("Timeout waiting for HTTP health check", "address", address)

			return fmt.Errorf("timeout waiting for HTTP health check %s", address)
		}

		if method != "" {
			method = http.MethodGet
		}

		buffBody := bytes.NewBuffer([]byte(body))

		rq, err := http.NewRequest(method, address, buffBody)
		if err != nil {
			return fmt.Errorf("unable to constrcut http request: %s", err)
		}

		rq.Header = headers

		if len(codes) == 0 {
			codes = []int{200}
		}

		resp, err := h.httpc.Do(rq)
		if err == nil && assertResponseCode(codes, resp.StatusCode) {
			h.l.Debug("HTTP health check complete", "address", address)
			return nil
		}

		// back off
		time.Sleep(h.backoff)
	}
}

func (h *HTTPImpl) HealthCheckTCP(address string, timeout time.Duration) error {
	h.l.Debug("Performing TCP health check for address", "address", address)
	st := time.Now()
	for {
		if time.Since(st) > timeout {
			h.l.Error("timeout waiting for TCP health check", "address", address)

			return fmt.Errorf("timeout waiting for HTTP health check %s", address)
		}

		// attempt to open a socket
		_, err := net.Dial("tcp", address)
		if err == nil {
			h.l.Debug("TCP health check complete", "address", address)
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
