package clients

import (
	"encoding/json"
	"os"
)

// NomadConfig defines a config file which is used to store Nomad cluster
// connection info
type NomadConfig struct {
	// Location of the Nomad cluster
	Location string `json:"location"`
	// Number of nodes in the cluster
	NodeCount int `json:"node_count"`
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
