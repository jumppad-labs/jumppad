package clients

import (
	"encoding/json"
	"os"

	"github.com/hashicorp/go-hclog"
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
}

// NewNomad creates a new Nomad client
func NewNomad(c HTTP, l hclog.Logger) Nomad {
	return &NomadImpl{c, l}
}

// SetConfig loads the Nomad config from a file
func (n *NomadImpl) SetConfig(nomadconfig string) error {
	return nil
}

// Apply the files to the nomad cluster and wait until all jobs are running
func (n *NomadImpl) Apply(files []string, waitUntilReady bool) error {
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
