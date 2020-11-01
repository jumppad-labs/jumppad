package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/hashicorp/go-hclog"
	"golang.org/x/xerrors"
)

// Nomad defines an interface for a Nomad client
type Nomad interface {
	// SetConfig for the client, path is a valid Nomad JSON config file
	SetConfig(string) error
	// Create jobs in the provided files
	Create(files []string) error
	// Stop jobs in the provided files
	Stop(files []string) error
	// ParseJob in the given file and return a JSON blob representing the HCL job
	ParseJob(file string) ([]byte, error)
	// JobStatus returns the status for the given job
	JobStatus(job string) (string, error)
	// HealthCheckAPI uses the Nomad API to check that all servers and nodes
	// are ready. The function will block until either all nodes are healthy or the
	// timeout period elapses.
	HealthCheckAPI(time.Duration) error
}

// NomadImpl is an implementation of the Nomad interface
type NomadImpl struct {
	httpClient HTTP
	l          hclog.Logger
	c          *ClusterConfig
	backoff    time.Duration
}

// NewNomad creates a new Nomad client
func NewNomad(c HTTP, backoff time.Duration, l hclog.Logger) Nomad {
	return &NomadImpl{httpClient: c, l: l, backoff: backoff}
}

type validateRequest struct {
	JobHCL       string
	Canonicalize bool
}

type createRequest struct {
	Job string
}

// SetConfig loads the Nomad config from a file
func (n *NomadImpl) SetConfig(nomadconfig string) error {
	c := &ClusterConfig{}
	err := c.Load(nomadconfig)
	if err != nil {
		return err
	}

	n.c = c

	return nil
}

// HealthCheckAPI executes a HTTP heathcheck for a Nomad cluster
func (n *NomadImpl) HealthCheckAPI(timeout time.Duration) error {
	// get the address and the nodecount from the config
	address := n.c.APIAddress()
	nodeCount := n.c.NodeCount

	n.l.Debug("Performing Nomad health check", "address", address)
	st := time.Now()
	for {
		if time.Now().Sub(st) > timeout {
			n.l.Error("Timeout wating for Nomad healthcheck", "address", address)

			return fmt.Errorf("Timeout waiting for Nomad healthcheck %s", address)
		}

		rq, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/v1/nodes", address), nil)
		if err != nil {
			return err
		}

		resp, err := n.httpClient.Do(rq)
		if err == nil && resp.StatusCode == 200 {
			nodes := []map[string]interface{}{}
			// check number of nodes
			json.NewDecoder(resp.Body).Decode(&nodes)

			// loop nodes and check ready
			readyCount := 0
			for _, node := range nodes {
				// get the node status
				nodeStatus := node["Status"].(string)
				nodeName := node["Name"].(string)
				nodeEligable := node["SchedulingEligibility"].(string)

				n.l.Debug("Node status", "node", nodeName, "status", nodeStatus, "eligible", nodeEligable)
				// get the driver status
				drivers, ok := node["Drivers"].(map[string]interface{})
				if !ok {
					continue
				}

				var driversHealthy = true
				for k, v := range drivers {
					driver, ok := v.(map[string]interface{})
					if !ok {
						continue
					}

					healthy, ok := driver["Healthy"].(bool)
					if !ok {
						continue
					}

					detected, ok := driver["Detected"].(bool)
					if !ok || !detected {
						continue
					}

					n.l.Debug("Driver status", "node", nodeName, "driver", k, "healthy", healthy)
					if healthy != true {
						driversHealthy = false
					}
				}

				if nodeStatus == "ready" && nodeEligable == "eligible" && driversHealthy {
					readyCount++
				}
			}

			if readyCount == nodeCount {
				n.l.Debug("Nomad check complete", "address", address)
				return nil
			}

			n.l.Debug("Nodes not ready", "ready", readyCount, "nodes", nodeCount)
		}

		// backoff
		time.Sleep(n.backoff)
	}
}

// Create jobs in the Nomad cluster for the given files and wait until all jobs are running
func (n *NomadImpl) Create(files []string) error {
	for _, f := range files {
		// parse the job
		jsonJob, err := n.ParseJob(f)
		if err != nil {
			return err
		}

		// submit the job top the API
		cr := fmt.Sprintf(`{"Job": %s}`, string(jsonJob))

		r, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/v1/jobs", n.c.APIAddress()), bytes.NewReader([]byte(cr)))
		if err != nil {
			return xerrors.Errorf("Unable to create http request: %w", err)
		}

		resp, err := n.httpClient.Do(r)
		if err != nil {
			return xerrors.Errorf("Unable to submit job: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			// try to read the body for the error
			d, _ := ioutil.ReadAll(resp.Body)
			return xerrors.Errorf("Error submitting job, got status code %d, error: %s", resp.StatusCode, string(d))
		}
	}

	return nil
}

// Stop the jobs defined in the files for the referenced Nomad cluster
func (n *NomadImpl) Stop(files []string) error {
	for _, f := range files {
		id, err := n.getJobID(f)
		if err != nil {
			return err
		}

		// stop the job
		r, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/v1/job/%s", n.c.APIAddress(), id), nil)
		if err != nil {
			return xerrors.Errorf("Unable to create http request: %w", err)
		}

		resp, err := n.httpClient.Do(r)
		if err != nil {
			return xerrors.Errorf("Unable to submit job: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return xerrors.Errorf("Error submitting job, got status code %d", resp.StatusCode)
		}
	}

	return nil
}

// ParseJob validates a HCL job file with the Nomad API and returns a slice of
// bytes representing the JSON payload.
func (n *NomadImpl) ParseJob(file string) ([]byte, error) {
	// load the file
	d, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, xerrors.Errorf("Unable to read file %s: %w", file, err)
	}

	// build the request object
	rd := validateRequest{
		JobHCL: string(d),
	}
	jobData, _ := json.Marshal(rd)

	// validate the config with the Nomad API
	r, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/v1/jobs/parse", n.c.APIAddress()), bytes.NewReader(jobData))
	if err != nil {
		return nil, xerrors.Errorf("Unable to create http request: %w", err)
	}

	resp, err := n.httpClient.Do(r)
	if err != nil {
		return nil, xerrors.Errorf("Unable to validate job: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, xerrors.Errorf("Error validating job, got status code %d", resp.StatusCode)
	}

	// job is valid submit to the server
	defer resp.Body.Close()
	jsonJob, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, xerrors.Errorf("Unable to read job from validate response: %w", err)
	}

	// return the job as a map
	return jsonJob, nil
}

// JobStatus returns a string status for the given job
func (n *NomadImpl) JobStatus(job string) (string, error) {
	// get the allocations for the job
	r, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/v1/job/%s", n.c.APIAddress(), job), nil)
	if err != nil {
		return "", xerrors.Errorf("Unable to create http request: %w", err)
	}

	resp, err := n.httpClient.Do(r)
	if err != nil {
		return "", xerrors.Errorf("Unable to validate job: %w", err)
	}
	defer resp.Body.Close()

	jobDetail := make(map[string]interface{}, 0)
	err = json.NewDecoder(resp.Body).Decode(&jobDetail)
	if err != nil {
		return "", err
	}

	return jobDetail["Status"].(string), nil
}

func (n *NomadImpl) getJobID(file string) (string, error) {
	// parse the job
	jsonJob, err := n.ParseJob(file)
	if err != nil {
		return "", err
	}

	// convert to a map to read the ID
	jobMap := make(map[string]interface{})
	err = json.Unmarshal(jsonJob, &jobMap)
	if err != nil {
		return "", err
	}

	return jobMap["ID"].(string), nil
}
