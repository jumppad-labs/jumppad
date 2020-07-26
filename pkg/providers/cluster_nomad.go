package providers

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"golang.org/x/xerrors"
)

const nomadBaseImage = "shipyardrun/nomad"
const nomadBaseVersion = "v0.11.2"

const dataDir = `
data_dir = "/etc/nomad.d/data"
`

const serverConfig = `
server {
  enabled = true
  bootstrap_expect = 1
}
`

const clientConfig = `
client {
	enabled = true

	server_join {
		retry_join = ["%s"]
	}
}

plugin "raw_exec" {
  config {
	enabled = true
  }
}
`

// NomadCluster defines a provider which can create Kubernetes clusters
type NomadCluster struct {
	config      *config.NomadCluster
	client      clients.ContainerTasks
	nomadClient clients.Nomad
	log         hclog.Logger
}

// NewNomadCluster creates a new Nomad cluster provider
func NewNomadCluster(c *config.NomadCluster, cc clients.ContainerTasks, hc clients.Nomad, l hclog.Logger) *NomadCluster {
	return &NomadCluster{c, cc, hc, l}
}

// Create implements interface method to create a cluster of the specified type
func (c *NomadCluster) Create() error {
	return c.createNomad()
}

// Destroy implements interface method to destroy a cluster
func (c *NomadCluster) Destroy() error {
	return c.destroyNomad()
}

// Lookup the a clusters current state
func (c *NomadCluster) Lookup() ([]string, error) {
	return c.client.FindContainerIDs(fmt.Sprintf("server.%s", c.config.Name), c.config.Type)
}

func (c *NomadCluster) createNomad() error {
	c.log.Info("Creating Cluster", "ref", c.config.Name)

	// check the client nodes do not already exist
	for i := 0; i < c.config.ClientNodes; i++ {
		ids, err := c.client.FindContainerIDs(fmt.Sprintf("%d.client.%s", i+1, c.config.Name), c.config.Type)
		if len(ids) > 0 {
			return fmt.Errorf("Client already exists")
		}

		if err != nil {
			return xerrors.Errorf("Unable to lookup cluster id: %w", err)
		}
	}

	// check the server does not already exist
	ids, err := c.client.FindContainerIDs(fmt.Sprintf("server.%s", c.config.Name), c.config.Type)
	if len(ids) > 0 {
		return ErrorClusterExists
	}

	if err != nil {
		return xerrors.Errorf("Unable to lookup cluster id: %w", err)
	}

	// if the version is not set use the default version
	if c.config.Version == "" {
		c.config.Version = nomadBaseVersion
	}

	// set the image
	image := fmt.Sprintf("%s:%s", nomadBaseImage, c.config.Version)

	// pull the container image
	err = c.client.PullImage(config.Image{Name: image}, false)
	if err != nil {
		return err
	}

	// create the volume for the cluster
	volID, err := c.client.CreateVolume(utils.ImageVolumeName)
	if err != nil {
		return err
	}

	isClient := true
	if c.config.ClientNodes > 0 {
		isClient = false
	}

	serverID, configDir, configPath, err := c.createServerNode(image, volID, isClient)
	if err != nil {
		return err
	}

	clients := []string{}
	clWait := sync.WaitGroup{}
	clWait.Add(c.config.ClientNodes)

	var clientError error
	for i := 0; i < c.config.ClientNodes; i++ {
		// create client node asyncronously
		go func(i int, image, volID, configDir, name string) {
			clientID, err := c.createClientNode(i, image, volID, configDir, name)
			if err != nil {
				clientError = err
			}

			clients = append(clients, clientID)
			clWait.Done()
		}(i+1, image, volID, configDir, utils.FQDN(fmt.Sprintf("server.%s", c.config.Name), string(config.TypeNomadCluster)))
	}

	clWait.Wait()
	if clientError != nil {
		return xerrors.Errorf("Unable to create client nodes: %w", clientError)
	}

	// ensure all client nodes are up
	c.nomadClient.SetConfig(configPath)
	err = c.nomadClient.HealthCheckAPI(startTimeout)
	if err != nil {
		return err
	}

	// import the images to the servers container d instance
	// importing images means that k3s does not need to pull from a remote docker hub
	if c.config.Images != nil && len(c.config.Images) > 0 {
		// import into the server
		err := c.ImportLocalDockerImages("images", serverID, c.config.Images, false)
		if err != nil {
			return xerrors.Errorf("Error importing Docker images: %w", err)
		}

		// import cached images to the clients asynchronously
		clWait := sync.WaitGroup{}
		clWait.Add(c.config.ClientNodes)
		var importErr error
		for _, id := range clients {
			go func(id string) {
				importErr = c.ImportLocalDockerImages("images", id, c.config.Images, false)
				clWait.Done()
				if err != nil {
					importErr = xerrors.Errorf("Error importing Docker images: %w", err)
				}
			}(id)
		}

		clWait.Wait()
		if importErr != nil {
			return importErr
		}
	}

	return nil
}

func (c *NomadCluster) createServerNode(image, volumeID string, isClient bool) (string, string, string, error) {
	// set the API server port to a random number 64000 - 65000
	apiPort := rand.Intn(1000) + 64000

	// if the node count is 0 we are creating a combo client server
	nodeCount := 1
	if c.config.ClientNodes > 0 {
		nodeCount = c.config.ClientNodes
	}

	// generate the config file
	nomadConfig := clients.NomadConfig{Location: fmt.Sprintf("http://localhost:%d", apiPort), NodeCount: nodeCount}
	configDir, configPath := utils.CreateNomadConfigPath(c.config.Name)

	err := nomadConfig.Save(configPath)
	if err != nil {
		return "", "", "", xerrors.Errorf("Unable to generate Nomad config: %w", err)
	}

	// generate the server config
	sc := dataDir + "\n" + serverConfig

	// if we have custom server config use that
	if c.config.ServerConfig != "" {
		sc = dataDir + "\n" + c.config.ServerConfig
	}

	// if the server also functions as a client
	if isClient {
		sc = sc + "\n" + fmt.Sprintf(clientConfig, "localhost")

		// if we have custom client config use that
		if c.config.ClientConfig != "" {
			sc = sc + c.config.ClientConfig
		}
	}

	// write the config to a file
	serverConfigPath := path.Join(configDir, "server_config.hcl")
	ioutil.WriteFile(serverConfigPath, []byte(sc), os.ModePerm)

	// create the server
	// since the server is just a container create the container config and provider
	cc := config.NewContainer(fmt.Sprintf("server.%s", c.config.Name))
	c.config.ResourceInfo.AddChild(cc)

	cc.Image = config.Image{Name: image}
	cc.Networks = c.config.Networks
	cc.Privileged = true // nomad must run Privileged as Docker needs to manipulate ip tables and stuff

	// set the volume mount for the images and the config
	cc.Volumes = []config.Volume{
		config.Volume{
			Source:      volumeID,
			Destination: "/images",
			Type:        "volume",
		},
		config.Volume{
			Source:      serverConfigPath,
			Destination: "/etc/nomad.d/config.hcl",
			Type:        "bind",
		},
	}

	// if there are any custom volumes to mount
	for _, v := range c.config.Volumes {
		cc.Volumes = append(cc.Volumes, v)
	}

	// set the environment variables for the K3S_KUBECONFIG_OUTPUT and K3S_CLUSTER_SECRET
	cc.Environment = c.config.Environment

	// expose the API server port
	cc.Ports = []config.Port{
		config.Port{
			Local:    "4646",
			Host:     fmt.Sprintf("%d", apiPort),
			Protocol: "tcp",
		},
	}

	id, err := c.client.CreateContainer(cc)
	if err != nil {
		return "", "", "", err
	}

	return id, configDir, configPath, nil
}

func (c *NomadCluster) createClientNode(index int, image, volumeID, configDir, serverID string) (string, error) {
	// generate the client config
	sc := dataDir + "\n" + fmt.Sprintf(clientConfig, serverID)

	// if we have custom config use that
	if c.config.ClientConfig != "" {
		sc = dataDir + "\n" + c.config.ClientConfig
	}

	// write the config to a file
	clientConfigPath := path.Join(configDir, "client_config.hcl")
	ioutil.WriteFile(clientConfigPath, []byte(sc), os.ModePerm)

	// create the server
	// since the server is just a container create the container config and provider
	cc := config.NewContainer(fmt.Sprintf("%d.client.%s", index, c.config.Name))
	c.config.ResourceInfo.AddChild(cc)

	cc.Image = config.Image{Name: image}
	cc.Networks = c.config.Networks
	cc.Privileged = true // nomad must run Privileged as Docker needs to manipulate ip tables and stuff

	// set the volume mount for the images and the config
	cc.Volumes = []config.Volume{
		config.Volume{
			Source:      volumeID,
			Destination: "/images",
			Type:        "volume",
		},
		config.Volume{
			Source:      clientConfigPath,
			Destination: "/etc/nomad.d/config.hcl",
			Type:        "bind",
		},
	}

	// if there are any custom volumes to mount
	for _, v := range c.config.Volumes {
		cc.Volumes = append(cc.Volumes, v)
	}

	// set the environment variables for the K3S_KUBECONFIG_OUTPUT and K3S_CLUSTER_SECRET
	cc.Environment = c.config.Environment

	return c.client.CreateContainer(cc)
}

// ImportLocalDockerImages fetches Docker images stored on the local client and imports them into the cluster
func (c *NomadCluster) ImportLocalDockerImages(name string, id string, images []config.Image, force bool) error {
	imgs := []string{}

	for _, i := range images {
		err := c.client.PullImage(i, false)
		if err != nil {
			return err
		}

		imgs = append(imgs, i.Name)
	}

	// import to volume
	vn := utils.FQDNVolumeName(name)
	imagesFile, err := c.client.CopyLocalDockerImageToVolume(imgs, vn, force)
	if err != nil {
		return err
	}

	// execute the command to import the image
	// write any command output to the logger
	for _, i := range imagesFile {
		err = c.client.ExecuteCommand(id, []string{"docker", "load", "-i", "/images/" + i}, nil, "/", c.log.StandardWriter(&hclog.StandardLoggerOptions{ForceLevel: hclog.Debug}))
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *NomadCluster) destroyNomad() error {
	c.log.Info("Destroy Nomad Cluster", "ref", c.config.Name)

	// destroy the server
	err := c.destroyNode(fmt.Sprintf("server.%s", c.config.Name))
	if err != nil {
		return err
	}

	// destroy the clients
	for i := 0; i < c.config.ClientNodes; i++ {
		err := c.destroyNode(fmt.Sprintf("%d.client.%s", i+1, c.config.Name))
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *NomadCluster) destroyNode(id string) error {
	// FindContainerIDs works on absolute addresses, we need to append the server
	ids, err := c.client.FindContainerIDs(id, c.config.Type)
	if err != nil {
		return err
	}

	for _, i := range ids {
		// remove from the networks
		for _, n := range c.config.Networks {
			c.log.Debug("Detaching container from network", "ref", c.config.Name, "id", i, "network", n.Name)
			err := c.client.DetachNetwork(n.Name, i)
			if err != nil {
				c.log.Error("Unable to detach network", "ref", c.config.Name, "network", n.Name, "error", err)
			}
		}

		err := c.client.RemoveContainer(i)
		if err != nil {
			return err
		}
	}

	return nil
}
