package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/hashicorp/go-hclog"
	"golang.org/x/xerrors"
)

// Nomad defines an interface for a Nomad client
type Nomad interface {
	SetConfig(string) error
	Create(files []string, waitUntilReady bool) error
	Stop(files []string) error
	ParseJob(file string) ([]byte, error)
	AllocationsRunning(file string) (map[string]bool, error)
}

// NomadImpl is an implementation of the Nomad interface
type NomadImpl struct {
	httpClient HTTP
	l          hclog.Logger
	c          *NomadConfig
}

// NewNomad creates a new Nomad client
func NewNomad(c HTTP, l hclog.Logger) Nomad {
	return &NomadImpl{httpClient: c, l: l}
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
	c := &NomadConfig{}
	err := c.Load(nomadconfig)
	if err != nil {
		return err
	}

	n.c = c

	return nil
}

// Create jobs in the Nomad cluster for the given files and wait until all jobs are running
func (n *NomadImpl) Create(files []string, waitUntilReady bool) error {
	for _, f := range files {
		// parse the job
		jsonJob, err := n.ParseJob(f)
		if err != nil {
			return err
		}

		// submit the job top the API
		cr := createRequest{Job: string(jsonJob)}
		crData, _ := json.Marshal(cr)

		r, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/v1/jobs", n.c.Location), bytes.NewReader(crData))
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

// Stop the jobs defined in the files for the referenced Nomad cluster
func (n *NomadImpl) Stop(files []string) error {
	for _, f := range files {
		id, err := n.getJobID(f)
		if err != nil {
			return err
		}

		// stop the job
		r, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/v1/job/%s", n.c.Location, id), nil)
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
	r, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/v1/jobs/parse", n.c.Location), bytes.NewReader(jobData))
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

// AllocationsRunning returns a map of allocations and if all the tasks within the job
// are running
func (n *NomadImpl) AllocationsRunning(file string) (map[string]bool, error) {
	id, err := n.getJobID(file)
	if err != nil {
		return nil, err
	}

	// get the allocations for the job
	r, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/v1/job/%s/allocations", n.c.Location, id), nil)
	if err != nil {
		return nil, xerrors.Errorf("Unable to create http request: %w", err)
	}

	resp, err := n.httpClient.Do(r)
	if err != nil {
		return nil, xerrors.Errorf("Unable to validate job: %w", err)
	}
	defer resp.Body.Close()

	allocs := make([]map[string]interface{}, 0)
	err = json.NewDecoder(resp.Body).Decode(&allocs)
	if err != nil {
		return nil, err
	}

	retData := map[string]bool{}

	for _, a := range allocs {
		running := true

		// check the status of all the tasks
		for _, t := range a["TaskStates"].(map[string]interface{}) {
			if t.(map[string]interface{})["State"].(string) != "running" {
				running = false
				break
			}
		}

		retData[a["ID"].(string)] = running
	}

	return retData, nil
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

// NomadConfig defines a config file which is used to store Nomad cluster
// connection info
type NomadConfig struct {
	// Location of the Nomad cluster
	Location string `json:"location"`
}

// Load the config from a file
func (n *NomadConfig) Load(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}

	return json.NewDecoder(f).Decode(n)
}

// Save the config to a file
func (n *NomadConfig) Save(file string) error {
	// if the file exists delete
	fs, err := os.Stat(file)
	if err != nil && fs != nil {
		err := os.Remove(file)
		if err != nil {
			return err
		}
	}

	f, err := os.Create(file)
	if err != nil {
		return err
	}

	return json.NewEncoder(f).Encode(n)
}
