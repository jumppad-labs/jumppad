package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"golang.org/x/xerrors"
)

// Nomad defines an interface for a Nomad client
type Nomad interface {
	// SetConfig for the client, path is a valid Nomad JSON config file
	SetConfig(utils.ClusterConfig, string) error
	// Create jobs in the provided files
	Create(files []string) error
	// Stop jobs in the provided files
	Stop(files []string) error
	// ParseJob in the given file and return a JSON blob representing the HCL job
	ParseJob(file string) ([]byte, error)
	// JobRunning returns true if all allocations for a job are running
	JobRunning(job string) (bool, error)
	// HealthCheckAPI uses the Nomad API to check that all servers and nodes
	// are ready. The function will block until either all nodes are healthy or the
	// timeout period elapses.
	HealthCheckAPI(time.Duration) error
	// Endpoints returns a list of endpoints for a cluster
	Endpoints(job, group, task string) ([]map[string]string, error)
}

// NomadImpl is an implementation of the Nomad interface
type NomadImpl struct {
	httpClient HTTP
	l          hclog.Logger
	c          *utils.ClusterConfig
	backoff    time.Duration
	context    string
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
func (n *NomadImpl) SetConfig(nomadconfig utils.ClusterConfig, context string) error {
	n.c = &nomadconfig
	n.context = context

	return nil
}

// HealthCheckAPI executes a HTTP heathcheck for a Nomad cluster
func (n *NomadImpl) HealthCheckAPI(timeout time.Duration) error {
	// get the address and the nodecount from the config
	address := n.c.APIAddress(utils.Context(n.context))
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
					if !healthy {
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

		r, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/v1/jobs", n.c.APIAddress(utils.Context(n.context))), bytes.NewReader([]byte(cr)))
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
		r, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/v1/job/%s", n.c.APIAddress(utils.Context(n.context)), id), nil)
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
	r, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/v1/jobs/parse", n.c.APIAddress(utils.Context(n.context))), bytes.NewReader(jobData))
	if err != nil {
		return nil, xerrors.Errorf("Unable to create http request: %w", err)
	}

	resp, err := n.httpClient.Do(r)
	if err != nil {
		return nil, xerrors.Errorf("Unable to validate job: %w", err)
	}

	defer resp.Body.Close()

	jsonJob, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, xerrors.Errorf("Unable to read response from Nomad API: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, xerrors.Errorf("Error validating job: %s", string(jsonJob))
	}

	// job is valid submit to the server
	// return the job as a map
	return jsonJob, nil
}

// JobRunning returns true when all allocations for a job are running
func (n *NomadImpl) JobRunning(job string) (bool, error) {
	jobDetail, err := n.getJobAllocations(job)
	if err != nil {
		return false, err
	}

	if len(jobDetail) < 1 {
		return false, nil
	}

	for _, v := range jobDetail {
		if v["ClientStatus"].(string) != "running" {
			return false, nil
		}
	}

	return true, nil
}

// Endpoints returns a list of endpoints for a cluster
func (n *NomadImpl) Endpoints(job, group, task string) ([]map[string]string, error) {
	jobs, err := n.getJobAllocations(job)
	if err != nil {
		return nil, err
	}

	endpoints := []map[string]string{}

	// get the allocation details for each endpoint
	for _, j := range jobs {
		r, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/v1/allocation/%s", n.c.APIAddress(utils.Context(n.context)), j["ID"]), nil)
		if err != nil {
			return nil, xerrors.Errorf("Unable to create http request: %w", err)
		}

		resp, err := n.httpClient.Do(r)
		if err != nil {
			return nil, xerrors.Errorf("Unable to get allocation: %w", err)
		}

		if resp.Body == nil {
			return nil, xerrors.Errorf("No body returned from Nomad API")
		}

		defer resp.Body.Close()

		allocDetail := allocation{}
		err = json.NewDecoder(resp.Body).Decode(&allocDetail)
		if err != nil {
			return nil, fmt.Errorf("Error getting endpoints from server: %s: err: %s", n.c.APIAddress(utils.Context(n.context)), err)
		}

		ports := []string{}

		// find the ports used by the task
		for _, tg := range allocDetail.Job.TaskGroups {
			if tg.Name == group {
				// non connect services will have their ports
				// coded in the driver config block
				for _, t := range tg.Tasks {
					if t.Name == task {
						ports = append(ports, t.Config.Ports...)
					}
				}

				// connect services will have this coded
				// in the groups network block
				for _, n := range tg.Networks {
					for _, dp := range n.DynamicPorts {
						ports = append(ports, dp.Label)
					}

					for _, dp := range n.ReservedPorts {
						ports = append(ports, dp.Label)
					}
				}
			}
		}

		ep := map[string]string{}
		epc := 0
		for _, p := range ports {
			// lookup the resources for the ports
			for _, n := range allocDetail.Resources.Networks {
				for _, dp := range n.DynamicPorts {
					if dp.Label == p {
						ep[p] = fmt.Sprintf("%s:%d", n.IP, dp.Value)
						epc++
					}
				}

				for _, dp := range n.ReservedPorts {
					if dp.Label == p {
						ep[p] = fmt.Sprintf("%s:%d", n.IP, dp.Value)
						epc++
					}
				}
			}
		}

		if epc > 0 {
			endpoints = append(endpoints, ep)
		}
	}

	return endpoints, nil
}

func (n *NomadImpl) getJobAllocations(job string) ([]map[string]interface{}, error) {
	// get the allocations for the job
	r, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/v1/job/%s/allocations", n.c.APIAddress(utils.Context(n.context)), job), nil)
	if err != nil {
		return nil, xerrors.Errorf("Unable to create http request: %w", err)
	}

	resp, err := n.httpClient.Do(r)
	if err != nil {
		return nil, xerrors.Errorf("Unable to query job: %w", err)
	}

	if resp.Body == nil {
		return nil, xerrors.Errorf("No body returned from Nomad API")
	}

	defer resp.Body.Close()

	jobDetail := make([]map[string]interface{}, 0)
	err = json.NewDecoder(resp.Body).Decode(&jobDetail)
	if err != nil {
		return nil, fmt.Errorf("Unable to query jobs in Nomad server: %s: %s", n.c.APIAddress(utils.Context(n.context)), err)
	}

	return jobDetail, err
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

type allocation struct {
	ID        string
	Job       job
	Resources resource
}

type job struct {
	Name       string
	TaskGroups []taskGroup
}

type taskGroup struct {
	Name     string
	Tasks    []task
	Networks []allocNetwork
}

type task struct {
	Name   string
	Config taskConfig
}

type taskConfig struct {
	Ports []string
}

type resource struct {
	Networks []allocNetwork
}

type allocNetwork struct {
	IP            string
	DynamicPorts  []port
	ReservedPorts []port
}

type port struct {
	Label string
	Value int
}
