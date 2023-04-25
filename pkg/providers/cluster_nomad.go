package providers

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/Masterminds/semver"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/hclconfig/types"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config/resources"
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
	config      *resources.NomadCluster
	client      clients.ContainerTasks
	nomadClient clients.Nomad
	log         hclog.Logger
}

// NewNomadCluster creates a new Nomad cluster provider
func NewNomadCluster(c *resources.NomadCluster, cc clients.ContainerTasks, hc clients.Nomad, l hclog.Logger) *NomadCluster {
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

	id, err := c.client.FindContainerIDs(utils.FQDN(fmt.Sprintf("server.%s", c.config.Name), c.config.Module, c.config.Type))
	if err != nil {
		return nil, err
	}

	ids = append(ids, id...)

	// find the clients
	for i := 0; i < c.config.ClientNodes; i++ {
		id, err := c.client.FindContainerIDs(utils.FQDN(fmt.Sprintf("%d.client.%s", i+1, c.config.Name), c.config.Module, c.config.Type))
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
		ids, err := c.client.FindContainerIDs(utils.FQDN(fmt.Sprintf("%d.client.%s", i+1, c.config.Name), c.config.Module, c.config.Type))
		if len(ids) > 0 {
			return fmt.Errorf("client already exists")
		}

		if err != nil {
			return xerrors.Errorf("unable to lookup cluster id: %w", err)
		}
	}

	// check the server does not already exist
	ids, err := c.client.FindContainerIDs(utils.FQDN(fmt.Sprintf("server.%s", c.config.Name), c.config.Module, c.config.Type))
	if len(ids) > 0 {
		return ErrClusterExists
	}

	if err != nil {
		return xerrors.Errorf("unable to lookup cluster id: %w", err)
	}

	// if the version is not set use the default version
	if c.config.Version == "" {
		c.config.Version = nomadBaseVersion
	}

	// set the image
	image := fmt.Sprintf("%s:%s", nomadBaseImage, c.config.Version)

	// pull the container image
	err = c.client.PullImage(resources.Image{Name: image}, false)
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

	c.config.APIPort = c.config.Port

	// set the API server port to a random number
	c.config.ConnectorPort = rand.Intn(utils.MaxRandomPort-utils.MinRandomPort) + utils.MinRandomPort
	c.config.ConfigDir = path.Join(utils.ShipyardHome(), c.config.Name, "config")
	c.config.ExternalIP = utils.GetDockerIP()

	serverID, err := c.createServerNode(image, volID, isClient)
	if err != nil {
		return err
	}

	name := fmt.Sprintf("server.%s", c.config.Name)
	c.config.ServerFQDN = utils.FQDN(name, c.config.Module, c.config.Type)

	cMutex := sync.Mutex{}
	clientFQDN := []string{}
	clientIDs := []string{}
	clWait := sync.WaitGroup{}
	clWait.Add(c.config.ClientNodes)

	var clientError error
	for i := 0; i < c.config.ClientNodes; i++ {
		// create client node asynchronously
		go func(i int, image, volID, name string) {
			clientID, err := c.createClientNode(i, image, volID, name)
			if err != nil {
				clientError = err
			}

			cMutex.Lock()

			clientIDs = append(clientIDs, clientID)
			clientName := fmt.Sprintf("%d.client.%s", i, c.config.Name)
			clientFQDN = append(clientFQDN, utils.FQDN(clientName, c.config.Module, c.config.Type))
			cMutex.Unlock()

			clWait.Done()
		}(i+1, image, volID, c.config.ServerFQDN)
	}

	clWait.Wait()
	if clientError != nil {
		return xerrors.Errorf("Unable to create client nodes: %w", clientError)
	}

	// set the client ids
	c.config.ClientFQDN = clientFQDN

	// if client nodes is 0 then the server acts as both client and server
	// in this instance set the health check to 1 node
	clientNodes := 1

	// otherwise use the number of specified client nodes
	if c.config.ClientNodes > 0 {
		clientNodes = c.config.ClientNodes
	}

	// ensure all client nodes are up
	c.nomadClient.SetConfig(fmt.Sprintf("http://%s", c.config.ExternalIP), c.config.APIPort, clientNodes)
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
		for _, id := range clientIDs {
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

func (c *NomadCluster) createServerNode(image, volumeID string, isClient bool) (string, error) {

	// generate the server config
	sc := dataDir + "\n" + serverConfig

	// if the server also functions as a client
	if isClient {
		sc = sc + "\n" + fmt.Sprintf(clientConfig, "localhost")
	}

	// write the nomad config to a file
	os.MkdirAll(c.config.ConfigDir, os.ModePerm)
	serverConfigPath := path.Join(c.config.ConfigDir, "server_config.hcl")
	ioutil.WriteFile(serverConfigPath, []byte(sc), os.ModePerm)

	// create the server
	// since the server is just a container create the container config and provider
	name := fmt.Sprintf("server.%s", c.config.Name)

	cc := &resources.Container{
		ResourceMetadata: types.ResourceMetadata{
			Name:   name,
			Module: c.config.Module,
			Type:   c.config.Type,
		},
	}

	cc.ParentConfig = c.config.ParentConfig

	cc.Image = &resources.Image{Name: image}
	cc.Networks = c.config.Networks
	cc.Privileged = true // nomad must run Privileged as Docker needs to manipulate ip tables and stuff

	// Add Consul DNS
	//cc.DNS = []string{"127.0.0.1"}

	// set the volume mount for the images and the config
	cc.Volumes = []resources.Volume{
		resources.Volume{
			Source:      volumeID,
			Destination: "/cache",
			Type:        "volume",
		},
		resources.Volume{
			Source:      serverConfigPath,
			Destination: "/etc/nomad.d/config.hcl",
			Type:        "bind",
		},
	}

	// Add any user config if set
	if c.config.ServerConfig != "" {
		vol := resources.Volume{
			Source:      c.config.ServerConfig,
			Destination: "/etc/nomad.d/server_user_config.hcl",
			Type:        "bind",
		}

		cc.Volumes = append(cc.Volumes, vol)
	}

	// Add any user config if set
	if c.config.ClientConfig != "" {
		vol := resources.Volume{
			Source:      c.config.ClientConfig,
			Destination: "/etc/nomad.d/client_user_config.hcl",
			Type:        "bind",
		}

		cc.Volumes = append(cc.Volumes, vol)
	}

	// Add the custom consul config if set
	if c.config.ConsulConfig != "" {
		vol := resources.Volume{
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
	cc.Ports = []resources.Port{
		resources.Port{
			Local:    "4646",
			Host:     fmt.Sprintf("%d", c.config.APIPort),
			Protocol: "tcp",
		},
		resources.Port{
			Local:    "19090",
			Host:     fmt.Sprintf("%d", c.config.ConnectorPort),
			Protocol: "tcp",
		},
	}

	cc.Environment = map[string]string{}
	err := c.appendProxyEnv(cc)
	if err != nil {
		return "", err
	}

	id, err := c.client.CreateContainer(cc)
	if err != nil {
		return "", err
	}

	return id, nil
}

func (c *NomadCluster) createClientNode(index int, image, volumeID, serverID string) (string, error) {
	// generate the client config
	sc := dataDir + "\n" + fmt.Sprintf(clientConfig, serverID)

	// write the default config to a file
	clientConfigPath := path.Join(c.config.ConfigDir, "client_config.hcl")
	ioutil.WriteFile(clientConfigPath, []byte(sc), os.ModePerm)

	// create the server
	// since the server is just a container create the container config and provider
	name := fmt.Sprintf("%d.client.%s", index, c.config.Name)
	cc := &resources.Container{
		ResourceMetadata: types.ResourceMetadata{
			Name:   name,
			Module: c.config.Module,
			Type:   c.config.Type,
		},
	}

	cc.ParentConfig = c.config.ParentConfig

	cc.Image = &resources.Image{Name: image}
	cc.Networks = c.config.Networks
	cc.Privileged = true // nomad must run Privileged as Docker needs to manipulate ip tables and stuff

	//cc.DNS = []string{"127.0.0.1"}

	// set the volume mount for the images and the config
	cc.Volumes = []resources.Volume{
		resources.Volume{
			Source:      volumeID,
			Destination: "/cache",
			Type:        "volume",
		},
		resources.Volume{
			Source:      clientConfigPath,
			Destination: "/etc/nomad.d/config.hcl",
			Type:        "bind",
		},
	}

	// Add any user config if set
	if c.config.ClientConfig != "" {
		vol := resources.Volume{
			Source:      c.config.ClientConfig,
			Destination: "/etc/nomad.d/user_config.hcl",
			Type:        "bind",
		}

		cc.Volumes = append(cc.Volumes, vol)
	}

	// Add the custom consul config if set
	if c.config.ConsulConfig != "" {
		vol := resources.Volume{
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

	cc.Environment = map[string]string{}
	err := c.appendProxyEnv(cc)
	if err != nil {
		return "", err
	}

	return c.client.CreateContainer(cc)
}

func (c *NomadCluster) appendProxyEnv(cc *resources.Container) error {

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

		cc.Environment["HTTP_PROXY"] = utils.HTTPProxyAddress()
		cc.Environment["HTTPS_PROXY"] = utils.HTTPSProxyAddress()
		cc.Environment["NO_PROXY"] = utils.ProxyBypass
		cc.Environment["PROXY_CA"] = string(ca)
	}

	return nil
}

// ImportLocalDockerImages fetches Docker images stored on the local client and imports them into the cluster
func (c *NomadCluster) ImportLocalDockerImages(name string, id string, images []resources.Image, force bool) error {
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
	os.RemoveAll(c.config.ConfigDir)

	return nil
}

func (c *NomadCluster) destroyNode(id string) error {
	// FindContainerIDs works on absolute addresses, we need to append the server
	ids, err := c.Lookup()
	if err != nil {
		return err
	}

	// fetch the networks
	nets := []types.Resource{}
	for _, n := range c.config.Networks {
		r, err := c.config.ParentConfig.FindResource(n.ID)
		if err != nil {
			c.log.Warn("Unable to find network", "id", n.ID, "error", err)
			continue
		}

		nets = append(nets, r)
	}

	for _, i := range ids {
		// remove from the networks
		for _, n := range nets {
			c.log.Debug("Detaching container from network", "ref", c.config.ID, "id", i, "network", n.Metadata().Name)
			err := c.client.DetachNetwork(n.Metadata().Name, i)
			if err != nil {
				c.log.Error("Unable to detach network", "ref", c.config.ID, "network", n.Metadata().Name, "error", err)
			}
		}

		err := c.client.RemoveContainer(i, false)
		if err != nil {
			return err
		}
	}

	return nil
}
