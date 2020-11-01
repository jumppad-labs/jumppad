package clients

import (
	"encoding/json"
	"fmt"
	"os"
)

// ClusterConfig defines a config file which is used to store Nomad cluster
// connection info
type ClusterConfig struct {
	// Location of the Cluster
	Address string `json:"address"`

	// Port the API Server is running on
	APIPort int `json:"api_port"`

	// Port where the connector is running
	ConnectorPort int `json:"connector_port"`

	// Number of nodes in the cluster
	NodeCount int `json:"node_count"`

	// Does the API use SSL?
	SSL bool `json:"ssl"`
}

// Load the config from a file
func (n *ClusterConfig) Load(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}

	return json.NewDecoder(f).Decode(n)
}

// Save the config to a file
func (n *ClusterConfig) Save(file string) error {
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
	defer f.Close()

	return json.NewEncoder(f).Encode(n)
}

// APIAddress returns the FQDN for the API server
func (n *ClusterConfig) APIAddress() string {
	protocol := "http"
	if n.SSL {
		protocol = "https"
	}

	return fmt.Sprintf("%s://%s:%d", protocol, n.Address, n.APIPort)
}

// ConnectorAddress returns the FQDN for the gRPC endpoing of the Connector
func (n *ClusterConfig) ConnectorAddress() string {
	return fmt.Sprintf("%s:%d", n.Address, n.ConnectorPort)
}
