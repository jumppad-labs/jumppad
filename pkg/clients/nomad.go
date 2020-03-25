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
	Apply(files []string, waitUntilReady bool) error
	Delete(files []string) error
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

// Apply the files to the nomad cluster and wait until all jobs are running
func (n *NomadImpl) Apply(files []string, waitUntilReady bool) error {
	for _, f := range files {
		// load the file
		d, err := ioutil.ReadFile(f)
		if err != nil {
			return xerrors.Errorf("Unable to read file %s: %w", f, err)
		}

		// build the request object
		rd := validateRequest{
			JobHCL: string(d),
		}
		jobData, _ := json.Marshal(rd)

		// validate the config with the Nomad API
		r, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/v1/jobs/parse", n.c.Location), bytes.NewReader(jobData))
		if err != nil {
			return xerrors.Errorf("Unable to create http request: %w", err)
		}

		resp, err := n.httpClient.Do(r)
		if err != nil {
			return xerrors.Errorf("Unable to validate job: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return xerrors.Errorf("Error validating job, got status code %d", resp.StatusCode)
		}

		// job is valid submit to the server
		defer resp.Body.Close()
		jsonJob, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return xerrors.Errorf("Unable to read job from validate response: %w", err)
		}

		cr := createRequest{Job: string(jsonJob)}
		crData, _ := json.Marshal(cr)

		r, err = http.NewRequest(http.MethodPost, fmt.Sprintf("%s/v1/jobs", n.c.Location), bytes.NewReader(crData))
		if err != nil {
			return xerrors.Errorf("Unable to create http request: %w", err)
		}

		resp, err = n.httpClient.Do(r)
		if err != nil {
			return xerrors.Errorf("Unable to submit job: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return xerrors.Errorf("Error submitting job, got status code %d", resp.StatusCode)
		}
	}

	return nil
}

// Delete the files to the nomad cluster and wait until all jobs are running
func (n *NomadImpl) Delete(files []string) error {
	return nil
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
