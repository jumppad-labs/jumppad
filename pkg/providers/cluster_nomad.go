package providers

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/Masterminds/semver"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"golang.org/x/xerrors"
)

const nomadBaseImage = "shipyardrun/nomad"
const nomadBaseVersion = "1.4.0"

const dataDir = `
data_dir = "/var/lib/nomad"
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
	ids := []string{}

	id, err := c.client.FindContainerIDs(fmt.Sprintf("server.%s", c.config.Name), c.config.Type)
	if err != nil {
		return nil, err
	}

	ids = append(ids, id...)

	// find the clients
	for i := 0; i < c.config.ClientNodes; i++ {
		id, err := c.client.FindContainerIDs(fmt.Sprintf("%d.client.%s", i+1, c.config.Name), c.config.Type)
		if err != nil {
			return nil, err
		}

		ids = append(ids, id...)
	}

	return ids, nil
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

	serverID, clusterConfig, configPath, err := c.createServerNode(image, volID, isClient)
	if err != nil {
		return err
	}

	cMutex := sync.Mutex{}
	cls := []string{}
	clWait := sync.WaitGroup{}
	clWait.Add(c.config.ClientNodes)

	var clientError error
	for i := 0; i < c.config.ClientNodes; i++ {
		// create client node asynchronously
		go func(i int, image, volID, configPath, name string) {
			clientID, err := c.createClientNode(i, image, volID, configPath, name)
			if err != nil {
				clientError = err
			}

			cMutex.Lock()
			cls = append(cls, clientID)
			cMutex.Unlock()

			clWait.Done()
		}(i+1, image, volID, configPath, utils.FQDN(fmt.Sprintf("server.%s", c.config.Name), string(config.TypeNomadCluster)))
	}

	clWait.Wait()
	if clientError != nil {
		return xerrors.Errorf("Unable to create client nodes: %w", clientError)
	}

	// ensure all client nodes are up
	c.nomadClient.SetConfig(clusterConfig, string(utils.LocalContext))
	err = c.nomadClient.HealthCheckAPI(startTimeout)
	if err != nil {
		return err
	}

	// import the images to the servers container d instance
	// importing images means that Nomad does not need to pull from a remote docker hub
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
		for _, id := range cls {
			go func(id string) {
				err := c.ImportLocalDockerImages("images", id, c.config.Images, false)
				clWait.Done()
				if err != nil {
					cMutex.Lock()
					importErr = xerrors.Errorf("Error importing Docker images: %w", err)
					cMutex.Unlock()
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

func (c *NomadCluster) createServerNode(image, volumeID string, isClient bool) (string, utils.ClusterConfig, string, error) {
	// if the node count is 0 we are creating a combo client server
	nodeCount := 1
	if c.config.ClientNodes > 0 {
		nodeCount = c.config.ClientNodes
	}

	conf, configDir := utils.GetClusterConfig(string(config.TypeNomadCluster) + "." + c.config.Name)

	// add the nodecount to the config and save
	conf.NodeCount = nodeCount
	conf.Save(filepath.Join(configDir, "config.json"))

	// generate the server config
	sc := dataDir + "\n" + serverConfig

	// if the server also functions as a client
	if isClient {
		sc = sc + "\n" + fmt.Sprintf(clientConfig, "localhost")
	}

	// write the nomad config to a file
	serverConfigPath := path.Join(configDir, "server_config.hcl")
	ioutil.WriteFile(serverConfigPath, []byte(sc), os.ModePerm)

	// create the server
	// since the server is just a container create the container config and provider
	cc := config.NewContainer(fmt.Sprintf("server.%s", c.config.Name))
	c.config.ResourceInfo.AddChild(cc)

	cc.Image = &config.Image{Name: image}
	cc.Networks = c.config.Networks
	cc.Privileged = true // nomad must run Privileged as Docker needs to manipulate ip tables and stuff

	// Add Consul DNS
	//cc.DNS = []string{"127.0.0.1"}

	// set the volume mount for the images and the config
	cc.Volumes = []config.Volume{
		config.Volume{
			Source:      volumeID,
			Destination: "/cache",
			Type:        "volume",
		},
		config.Volume{
			Source:      serverConfigPath,
			Destination: "/etc/nomad.d/config.hcl",
			Type:        "bind",
		},
	}

	// Add any user config if set
	if c.config.ServerConfig != "" {
		vol := config.Volume{
			Source:      c.config.ServerConfig,
			Destination: "/etc/nomad.d/server_user_config.hcl",
			Type:        "bind",
		}

		cc.Volumes = append(cc.Volumes, vol)
	}

	// Add any user config if set
	if c.config.ClientConfig != "" {
		vol := config.Volume{
			Source:      c.config.ClientConfig,
			Destination: "/etc/nomad.d/client_user_config.hcl",
			Type:        "bind",
		}

		cc.Volumes = append(cc.Volumes, vol)
	}

	// Add the custom consul config if set
	if c.config.ConsulConfig != "" {
		vol := config.Volume{
			Source:      c.config.ConsulConfig,
			Destination: "/etc/consul.d/config/user_config.hcl",
			Type:        "bind",
		}

		cc.Volumes = append(cc.Volumes, vol)
	}

	// if there are any custom volumes to mount
	for _, v := range c.config.Volumes {
		cc.Volumes = append(cc.Volumes, v)
	}

	cc.Environment = c.config.Environment

	// expose the API server port
	cc.Ports = []config.Port{
		config.Port{
			Local:    "4646",
			Host:     fmt.Sprintf("%d", conf.APIPort),
			Protocol: "tcp",
		},
		config.Port{
			Local:    "19090",
			Host:     fmt.Sprintf("%d", conf.ConnectorPort),
			Protocol: "tcp",
		},
	}

	cc.EnvVar = map[string]string{}
	err := c.appendProxyEnv(cc)
	if err != nil {
		return "", utils.ClusterConfig{}, "", err
	}

	id, err := c.client.CreateContainer(cc)
	if err != nil {
		return "", utils.ClusterConfig{}, "", err
	}

	return id, conf, configDir, nil
}

func (c *NomadCluster) createClientNode(index int, image, volumeID, configDir, serverID string) (string, error) {
	// generate the client config
	sc := dataDir + "\n" + fmt.Sprintf(clientConfig, serverID)

	// write the default config to a file
	clientConfigPath := path.Join(configDir, "client_config.hcl")
	ioutil.WriteFile(clientConfigPath, []byte(sc), os.ModePerm)

	// create the server
	// since the server is just a container create the container config and provider
	cc := config.NewContainer(fmt.Sprintf("%d.client.%s", index, c.config.Name))
	c.config.ResourceInfo.AddChild(cc)

	cc.Image = &config.Image{Name: image}
	cc.Networks = c.config.Networks
	cc.Privileged = true // nomad must run Privileged as Docker needs to manipulate ip tables and stuff

	//cc.DNS = []string{"127.0.0.1"}

	// set the volume mount for the images and the config
	cc.Volumes = []config.Volume{
		config.Volume{
			Source:      volumeID,
			Destination: "/cache",
			Type:        "volume",
		},
		config.Volume{
			Source:      clientConfigPath,
			Destination: "/etc/nomad.d/config.hcl",
			Type:        "bind",
		},
	}

	// Add any user config if set
	if c.config.ClientConfig != "" {
		vol := config.Volume{
			Source:      c.config.ClientConfig,
			Destination: "/etc/nomad.d/user_config.hcl",
			Type:        "bind",
		}

		cc.Volumes = append(cc.Volumes, vol)
	}

	// Add the custom consul config if set
	if c.config.ConsulConfig != "" {
		vol := config.Volume{
			Source:      c.config.ConsulConfig,
			Destination: "/etc/consul.d/config/user_config.hcl",
			Type:        "bind",
		}

		cc.Volumes = append(cc.Volumes, vol)
	}

	// if there are any custom volumes to mount
	for _, v := range c.config.Volumes {
		cc.Volumes = append(cc.Volumes, v)
	}

	cc.Environment = c.config.Environment

	cc.EnvVar = map[string]string{}
	err := c.appendProxyEnv(cc)
	if err != nil {
		return "", err
	}

	return c.client.CreateContainer(cc)
}

func (c *NomadCluster) appendProxyEnv(cc *config.Container) error {

	// only add the variables for the cache when the nomad version is >= v0.11.8 or
	// is using the dev version
	usesCache := false
	if c.config.Version == "dev" {
		usesCache = true
	} else {

		sv, err := semver.NewConstraint(">= v0.11.8")
		if err != nil {
			// Handle constraint not being parsable.
			return err
		}

		v, err := semver.NewVersion(c.config.Version)
		if err != nil {
			return fmt.Errorf("Nomad version is not valid semantic version: %s", err)
		}

		usesCache = sv.Check(v)
	}

	if usesCache {
		// load the CA from a file
		ca, err := ioutil.ReadFile(filepath.Join(utils.CertsDir(""), "/root.cert"))
		if err != nil {
			return fmt.Errorf("Unable to read root CA for proxy: %s", err)
		}

		cc.EnvVar["HTTP_PROXY"] = utils.HTTPProxyAddress()
		cc.EnvVar["HTTPS_PROXY"] = utils.HTTPSProxyAddress()
		cc.EnvVar["NO_PROXY"] = utils.ProxyBypass
		cc.EnvVar["PROXY_CA"] = string(ca)
	}

	return nil
}

// ImportLocalDockerImages fetches Docker images stored on the local client and imports them into the cluster
func (c *NomadCluster) ImportLocalDockerImages(name string, id string, images []config.Image, force bool) error {
	imgs := []string{}

	for _, i := range images {
		// ignore when the name is empty
		if i.Name == "" {
			continue
		}

		err := c.client.PullImage(i, false)
		if err != nil {
			return err
		}

		imgs = append(imgs, i.Name)
	}

	// import to volume
	vn := utils.FQDNVolumeName(name)
	imagesFile, err := c.client.CopyLocalDockerImagesToVolume(imgs, vn, force)
	if err != nil {
		return err
	}

	// execute the command to import the image
	// write any command output to the logger
	for _, i := range imagesFile {
		err = c.client.ExecuteCommand(id, []string{"docker", "load", "-i", i}, nil, "/", "", "", c.log.StandardWriter(&hclog.StandardLoggerOptions{ForceLevel: hclog.Debug}))
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

	// remove the config
	_, path := utils.GetClusterConfig(string(c.config.Type) + "." + c.config.Name)
	os.RemoveAll(path)

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

		err := c.client.RemoveContainer(i, false)
		if err != nil {
			return err
		}
	}

	return nil
}
