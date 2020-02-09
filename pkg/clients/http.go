package clients

import (
	"encoding/json"
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
	// HealthCheckNomad uses the Nomad API to check that all servers and nodes
	// are ready. The function will block until either all nodes are healthy or the
	// timeout period elapses.
	HealthCheckNomad(api_addr string, nodeCount int, timeout time.Duration) error
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

func (h *HTTPImpl) HealthCheckNomad(address string, nodeCount int, timeout time.Duration) error {
	h.l.Debug("Performing Nomad health check for address", "address", address)
	st := time.Now()
	for {
		if time.Now().Sub(st) > timeout {
			h.l.Error("Timeout wating for Nomad healthcheck", "address", address)

			return fmt.Errorf("Timeout waiting for Nomad healthcheck %s", address)
		}

		resp, err := http.Get(fmt.Sprintf("%s/v1/nodes", address))
		if err == nil && resp.StatusCode == 200 {
			nodes := []map[string]interface{}{}
			// check number of nodes
			json.NewDecoder(resp.Body).Decode(&nodes)

			// loop nodes and check ready
			readyCount := 0
			for _, n := range nodes {
				if n["Status"].(string) == "ready" {
					readyCount++
				}
			}

			if readyCount == nodeCount {
				h.l.Debug("Nomad check complete", "address", address)
				return nil
			}
		}

		// backoff
		time.Sleep(h.backoff)
	}
}
