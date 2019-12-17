package providers

import (
	"fmt"
	"time"

	"github.com/shipyard-run/cli/pkg/clients"
)

// healthCheckPods uses the given selector to check that all pods are started
// and running.
// selectors are checked sequentially
// pods = ["component=server,app=consul", "component=client,app=consul"]
func healthCheckPods(client clients.Kubernetes, selectors []string, timeout time.Duration) error {
	// check all pods are running
	for _, s := range selectors {
		err := healthCheckSingle(client, s, timeout)
		if err != nil {
			return err
		}
	}

	return nil
}

// healthCheckSingle checks for running containers with the given selector
func healthCheckSingle(c clients.Kubernetes, selector string, timeout time.Duration) error {
	st := time.Now()
	for {
		if time.Now().Sub(st) > timeout {
			return fmt.Errorf("Timeout waiting for pods %s to start", selector)
		}

		// GetPods may return an error if the API server is not available
		pl, err := c.GetPods(selector)
		if err != nil {
			fmt.Println(err)
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
			break
		}

		// backoff
		time.Sleep(2 * time.Second)
	}

	return nil
}
