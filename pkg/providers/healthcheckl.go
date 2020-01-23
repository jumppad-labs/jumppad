package providers

import (
	"fmt"
	"net/http"
	"time"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
)

// healthCheckPods uses the given selector to check that all pods are started
// and running.
// selectors are checked sequentially
// pods = ["component=server,app=consul", "component=client,app=consul"]
func healthCheckPods(client clients.Kubernetes, selectors []string, timeout time.Duration, l hclog.Logger) error {
	// check all pods are running
	for _, s := range selectors {
		l.Debug("Performing health check for pod", "selector", s)
		err := healthCheckSingle(client, s, timeout, l)
		if err != nil {
			return err
		}
	}

	return nil
}

// healthCheckSingle checks for running containers with the given selector
func healthCheckSingle(c clients.Kubernetes, selector string, timeout time.Duration, l hclog.Logger) error {
	st := time.Now()
	for {
		if time.Now().Sub(st) > timeout {
			l.Error("Timeout wating for pod to start", "selector", selector)

			return fmt.Errorf("Timeout waiting for pods %s to start", selector)
		}

		// GetPods may return an error if the API server is not available
		pl, err := c.GetPods(selector)
		if err != nil {
			continue
		}

		// there should be at least 1 pod
		if len(pl.Items) < 1 {
			continue
		}

		allRunning := true
		for _, pod := range pl.Items {
			if pod.Status.Phase != "Running" {
				allRunning = false
				break
			}
		}

		if allRunning {
			l.Debug("Health check complete", "selector", selector)
			break
		}

		// backoff
		time.Sleep(2 * time.Second)
	}

	return nil
}

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

	return nil
}
