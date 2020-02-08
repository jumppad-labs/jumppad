package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	hclog "github.com/hashicorp/go-hclog"
)

func healthCheckHTTP(address string, timeout time.Duration, l hclog.Logger) error {
	l.Debug("Performing health check for address", "address", address)
	st := time.Now()
	for {
		if time.Now().Sub(st) > timeout {
			l.Error("Timeout wating for HTTP healthcheck", "address", address)

			return fmt.Errorf("Timeout waiting for HTTP healthcheck %s", address)
		}

		resp, err := http.Get(address)
		if err == nil && resp.StatusCode == 200 {
			l.Debug("Health check complete", "address", address)
			return nil
		}

		// backoff
		time.Sleep(2 * time.Second)
	}
}

func healthCheckNomad(address string, nodeCount int, timeout time.Duration, l hclog.Logger) error {
	l.Debug("Performing Nomad health check for address", "address", address)
	st := time.Now()
	for {
		if time.Now().Sub(st) > timeout {
			l.Error("Timeout wating for Nomad healthcheck", "address", address)

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
				l.Debug("Nomad check complete", "address", address)
				return nil
			}
		}

		// backoff
		time.Sleep(2 * time.Second)
	}
}
