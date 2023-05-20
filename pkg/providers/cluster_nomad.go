package providers

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/shipyard-run/hclconfig/types"
	"golang.org/x/xerrors"
)

// NomadCluster defines a provider which can create Kubernetes clusters
type NomadCluster struct {
	config      *resources.NomadCluster
	client      clients.ContainerTasks
	nomadClient clients.Nomad
	connector   clients.Connector
	log         hclog.Logger
}

// NewNomadCluster creates a new Nomad cluster provider
func NewNomadCluster(c *resources.NomadCluster, cc clients.ContainerTasks, hc clients.Nomad, con clients.Connector, l hclog.Logger) *NomadCluster {
	return &NomadCluster{c, cc, hc, con, l}
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

	id, err := c.client.FindContainerIDs(c.config.ServerFQRN)
	if err != nil {
		return nil, err
	}

	ids = append(ids, id...)

	// find the clients
	for _, id := range c.config.ClientFQRN {
		id, err := c.client.FindContainerIDs(id)
		if err != nil {
			return nil, err
		}

		ids = append(ids, id...)
	}

	return ids, nil
}

// Refresh is called when `up` is run and the resource has been marked as created
// checks the nodes are healthy and replaces if needed.
func (c *NomadCluster) Refresh() error {
	c.log.Info("Refresh Nomad Cluster", "ref", c.config.ID)

	c.log.Debug("Checking health of server node", "ref", c.config.ID, "server", c.config.ServerFQRN)

	ids, _ := c.client.FindContainerIDs(c.config.ServerFQRN)
	if len(ids) == 1 {
		c.log.Debug("Server node exists", "ref", c.config.ID, "server", c.config.ServerFQRN, "id", ids[0])
	}

	// find any nodes that have crashed or have been deleted
	for _, cl := range c.config.ClientFQRN {
		c.log.Debug("Checking health of client nodes", "ref", c.config.ID, "client", cl)

		ids, _ := c.client.FindContainerIDs(cl)
		if len(ids) == 1 {
			c.log.Debug("Client node healthy", "ref", c.config.ID, "client", cl, "id", ids[0])
		} else {
			c.log.Debug("Client node does not exist", "ref", c.config.ID, "client", cl)
			// recreate the node
			c.config.ClientFQRN = removeElement(c.config.ClientFQRN, cl)
		}
	}

	// Has the number of clients nodes changed and are we scaling down?
	if c.config.ClientNodes < len(c.config.ClientFQRN) {
		// calculate the number of nodes that should be removed
		removeCount := len(c.config.ClientFQRN) - c.config.ClientNodes
		c.log.Info("Scaling cluster down", "ref", c.config.ID, "current_scale", len(c.config.ClientFQRN), "new_scale", c.config.ClientNodes, "removing", removeCount)

		// add the nodes to the remove list
		nodesToRemove := c.config.ClientFQRN[:removeCount]

		wg := sync.WaitGroup{}
		wg.Add(len(nodesToRemove))

		for _, n := range nodesToRemove {
			c.log.Debug("Removing node", "ref", c.config.ID, "client", n)

			go func(name string) {
				err := c.destroyNode(name)
				wg.Done()

				if err != nil {
					c.log.Error("Unable to remove node", "ref", c.config.ID, "client", name)
				}
			}(n)

			c.config.ClientFQRN = removeElement(c.config.ClientFQRN, n)
		}

		wg.Wait()

		c.nomadClient.SetConfig(fmt.Sprintf("http://%s", c.config.ExternalIP), c.config.APIPort, c.config.ClientNodes+1)
		err := c.nomadClient.HealthCheckAPI(startTimeout)
		if err != nil {
			return err
		}

		return nil
	}

	// do we need to scale the cluster up
	if c.config.ClientNodes > len(c.config.ClientFQRN) {
		// need to scale up
		c.log.Info("Scaling cluster up", "ref", c.config.ID, "current_scale", len(c.config.ClientFQRN), "new_scale", c.config.ClientNodes)

		for i := len(c.config.ClientFQRN); i < c.config.ClientNodes; i++ {
			id := utils.FQDN(fmt.Sprintf("%s.client.%s", randomID(), c.config.Name), c.config.Module, c.config.Type)

			c.log.Debug("Create client node", "ref", c.config.ID, "client", id)

			fqdn, _, err := c.createClientNode(randomID(), c.config.Image.Name, utils.ImageVolumeName, c.config.ServerFQRN)
			if err != nil {
				return fmt.Errorf(`unable to recreate client node "%s", %s`, id, err)
			}

			c.config.ClientFQRN = append(c.config.ClientFQRN, fqdn)

			c.log.Debug("Successfully created client node", "ref", c.config.ID, "client", fqdn)
		}

		c.nomadClient.SetConfig(fmt.Sprintf("http://%s", c.config.ExternalIP), c.config.APIPort, c.config.ClientNodes+1)
		err := c.nomadClient.HealthCheckAPI(startTimeout)
		if err != nil {
			return err
		}

	}

	return nil
}

func removeElement(s []string, item string) []string {
	// find the element
	index := -1
	for i, f := range s {
		if f == item {
			index = i
			break
		}
	}

	// not found
	if index < 0 {
		return s
	}

	return append(s[:index], s[index+1:]...)
}

func (c *NomadCluster) createNomad() error {
	c.log.Info("Creating Cluster", "ref", c.config.ID)

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

	// pull the container image
	err = c.client.PullImage(*c.config.Image, false)
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

	// set the API server port to a random number
	c.config.ConnectorPort = rand.Intn(utils.MaxRandomPort-utils.MinRandomPort) + utils.MinRandomPort
	c.config.ConfigDir = path.Join(utils.JumppadHome(), c.config.Name, "config")
	c.config.ExternalIP = utils.GetDockerIP()

	serverID, err := c.createServerNode(c.config.Image.Name, volID, isClient)
	if err != nil {
		return err
	}

	name := fmt.Sprintf("server.%s", c.config.Name)
	c.config.ServerFQRN = utils.FQDN(name, c.config.Module, c.config.Type)

	cMutex := sync.Mutex{}
	clientFQDN := []string{}
	clientIDs := []string{}
	clWait := sync.WaitGroup{}
	clWait.Add(c.config.ClientNodes)

	var clientError error
	for i := 0; i < c.config.ClientNodes; i++ {
		// create client node asynchronously
		go func(id string, image, volID, name string) {
			fqdn, clientID, err := c.createClientNode(id, image, volID, name)
			if err != nil {
				clientError = err
			}

			cMutex.Lock()

			clientIDs = append(clientIDs, clientID)
			clientFQDN = append(clientFQDN, fqdn)
			cMutex.Unlock()

			clWait.Done()
		}(randomID(), c.config.Image.Name, volID, c.config.ServerFQRN)
	}

	clWait.Wait()

	// set the client ids
	c.config.ClientFQRN = clientFQDN

	if clientError != nil {
		return xerrors.Errorf("Unable to create client nodes: %w", clientError)
	}

	// if client nodes is 0 then the server acts as both client and server
	// in this instance set the health check to 1 node
	clientNodes := 1

	// otherwise use the number of specified client nodes
	if c.config.ClientNodes > 0 {
		clientNodes = c.config.ClientNodes + 1
	}

	// ensure all client nodes are up
	c.nomadClient.SetConfig(fmt.Sprintf("http://%s", c.config.ExternalIP), c.config.APIPort, clientNodes)
	err = c.nomadClient.HealthCheckAPI(startTimeout)
	if err != nil {
		return err
	}

	// import the images to the servers container d instance
	// importing images means that Nomad does not need to pull from a remote docker hub
	if c.config.CopyImages != nil && len(c.config.CopyImages) > 0 {
		// import into the server
		err := c.ImportLocalDockerImages("images", serverID, c.config.CopyImages, false)
		if err != nil {
			return xerrors.Errorf("Error importing Docker images: %w", err)
		}

		// import cached images to the clients asynchronously
		clWait := sync.WaitGroup{}
		clWait.Add(c.config.ClientNodes)
		var importErr error
		for _, id := range clientIDs {
			go func(id string) {
				err := c.ImportLocalDockerImages("images", id, c.config.CopyImages, false)
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

	err = c.deployConnector()
	if err != nil {
		return fmt.Errorf("unable to deploy Connector: %s", err)
	}

	return nil
}

func (c *NomadCluster) createServerNode(image, volumeID string, isClient bool) (string, error) {

	// set the resources for CPU, if not a client set the resources low
	// so that we can only deploy the connector to the server
	cpu := ""
	if !isClient {
		cpu = "cpu_total_compute = 500"
	}

	// generate the server config
	sc := dataDir + "\n" + fmt.Sprintf(serverConfig, cpu)

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
			Local:    fmt.Sprintf("%d", c.config.ConnectorPort),
			Host:     fmt.Sprintf("%d", c.config.ConnectorPort),
			Protocol: "tcp",
		},
		resources.Port{
			Local:    fmt.Sprintf("%d", c.config.ConnectorPort+1),
			Host:     fmt.Sprintf("%d", c.config.ConnectorPort+1),
			Protocol: "tcp",
		},
	}

	cc.Ports = append(cc.Ports, c.config.Ports...)
	cc.PortRanges = append(cc.PortRanges, c.config.PortRanges...)

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

// createClient node creates a Nomad client node
// returns the fqdn, docker id, and an error if unsuccessful
func (c *NomadCluster) createClientNode(id string, image, volumeID, serverID string) (string, string, error) {
	// generate the client config
	sc := dataDir + "\n" + fmt.Sprintf(clientConfig, serverID)

	// write the default config to a file
	clientConfigPath := path.Join(c.config.ConfigDir, "client_config.hcl")
	ioutil.WriteFile(clientConfigPath, []byte(sc), os.ModePerm)

	// create the server
	// since the server is just a container create the container config and provider
	name := fmt.Sprintf("%s.client.%s", id, c.config.Name)
	cc := &resources.Container{
		ResourceMetadata: types.ResourceMetadata{
			Name:   name,
			Module: c.config.Module,
			Type:   c.config.Type,
		},
	}

	fqdn := utils.FQDN(name, c.config.Module, c.config.Type)

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
	cc.Volumes = append(cc.Volumes, c.config.Volumes...)

	cc.Environment = c.config.Environment

	cc.Environment = map[string]string{}
	err := c.appendProxyEnv(cc)
	if err != nil {
		return "", "", err
	}

	cid, err := c.client.CreateContainer(cc)
	return fqdn, cid, err
}

func (c *NomadCluster) appendProxyEnv(cc *resources.Container) error {
	// load the CA from a file
	ca, err := ioutil.ReadFile(filepath.Join(utils.CertsDir(""), "/root.cert"))
	if err != nil {
		return fmt.Errorf("unable to read root CA for proxy: %s", err)
	}

	// add the netmask from the network to the proxy bypass
	networkSubmasks :=
		[]string{}
	for _, n := range c.config.Networks {
		net, err := c.config.ParentConfig.FindResource(n.ID)
		if err != nil {
			return fmt.Errorf("Network not found: %w", err)
		}

		networkSubmasks = append(networkSubmasks, net.(*resources.Network).Subnet)
	}

	proxyBypass := utils.ProxyBypass + "," + strings.Join(networkSubmasks, ",")

	cc.Environment["HTTP_PROXY"] = utils.HTTPProxyAddress()
	cc.Environment["HTTPS_PROXY"] = utils.HTTPSProxyAddress()
	cc.Environment["NO_PROXY"] = proxyBypass
	cc.Environment["PROXY_CA"] = string(ca)

	return nil
}

func (c *NomadCluster) deployConnector() error {
	c.log.Debug("Deploying connector", "ref", c.config.ID)

	// generate the certificates
	// generate the certificates for the service
	cb, err := c.connector.GetLocalCertBundle(utils.CertsDir(""))
	if err != nil {
		return fmt.Errorf("unable to fetch root certificates for ingress: %s", err)
	}

	// generate the leaf certificates ensuring that we add
	// the ip address for the docker hosts as this might not be local
	lf, err := c.connector.GenerateLeafCert(
		cb.RootKeyPath,
		cb.RootCertPath,
		[]string{
			"connector",
			fmt.Sprintf("%s:%d", utils.GetDockerIP(), c.config.ConnectorPort),
		},
		[]string{utils.GetDockerIP()},
		utils.CertsDir(c.config.ID),
	)

	// load the certs into a string so that they can be embedded into the config
	ca, _ := ioutil.ReadFile(lf.RootCertPath)
	cert, _ := ioutil.ReadFile(lf.LeafCertPath)
	key, _ := ioutil.ReadFile(lf.LeafKeyPath)

	if err != nil {
		return fmt.Errorf("unable to generate leaf certificates for ingress: %s", err)
	}

	// create a temp directory to write config to
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return fmt.Errorf("unable to create temporary directory: %s", err)
	}

	defer os.RemoveAll(dir)

	ll := os.Getenv("LOG_LEVEL")
	if ll == "" {
		ll = "info"
	}

	config := fmt.Sprintf(
		nomadConnectorDeployment,
		c.config.ConnectorPort,
		c.config.ConnectorPort+1,
		string(cert),
		string(key),
		string(ca),
		ll,
	)

	connectorDeployment := filepath.Join(dir, "connector.nomad")
	ioutil.WriteFile(connectorDeployment, []byte(config), os.ModePerm)

	// deploy the file
	err = c.nomadClient.Create([]string{connectorDeployment})
	if err != nil {
		return fmt.Errorf("unable to run Connector deployment: %s", err)
	}

	// wait until healthy
	timeout, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	var ok bool
	var lastError error

	for {
		if timeout.Err() != nil {
			break
		}

		ok, lastError = c.nomadClient.JobRunning("connector")
		if err != nil {
			lastError = fmt.Errorf("unable to check Connector deployment health: %s", err)
			continue
		}

		if ok {
			break
		}

		lastError = fmt.Errorf("Connector not healthy")
	}

	return lastError
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

	// destroy the clients
	wg := sync.WaitGroup{}
	wg.Add(len(c.config.ClientFQRN))

	for _, cl := range c.config.ClientFQRN {
		go func(name string) {
			defer wg.Done()

			err := c.destroyNode(name)
			if err != nil {
				c.log.Error("unable to remove cluster client", "client", name)
			}
		}(cl)
	}

	wg.Wait()

	// destroy the server
	err := c.destroyNode(c.config.ServerFQRN)
	if err != nil {
		return err
	}

	// remove the config
	os.RemoveAll(c.config.ConfigDir)

	return nil
}

func (c *NomadCluster) destroyNode(id string) error {
	// FindContainerIDs works on absolute addresses, we need to append the server
	ids, _ := c.client.FindContainerIDs(id)
	if len(ids) == 0 {
		// nothing to do
		return nil
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

func randomID() string {
	id := uuid.New()
	short := strings.Replace(id.String(), "-", "", -1)
	return short[:8]
}

var nomadConnectorDeployment = `
job "connector" {
  datacenters = ["dc1"]
  type        = "service"


  update {
    max_parallel      = 1
    min_healthy_time  = "10s"
    healthy_deadline  = "3m"
    progress_deadline = "10m"
    auto_revert       = false
    canary            = 0
  }

  migrate {
    max_parallel     = 1
    health_check     = "checks"
    min_healthy_time = "10s"
    healthy_deadline = "5m"
  }

  group "connector" {
    count = 1

    network {
      port "grpc" {
        to     = 60000
        static = %d
      }

      port "http" {
        to     = 60001
        static = %d
      }
    }

    restart {
      # The number of attempts to run the job within the specified interval.
      attempts = 2
      interval = "30m"
      delay    = "15s"
      mode     = "fail"
    }

    ephemeral_disk {
      size = 30
    }

    task "connector" {
  		constraint {
  		  attribute = "${meta.node_type}"
				operator  = "=" 
  		  value     = "server"
  		}

      template {
        data = <<-EOH
%s
        EOH

        destination = "local/certs/server.cert"
      }
      
      template {
        data = <<-EOH
%s
        EOH

        destination = "local/certs/server.key"
      }
      
      template {
        data = <<-EOH
%s
        EOH

        destination = "local/certs/ca.cert"
      }

      # The "driver" parameter specifies the task driver that should be used to
      # run the task.
      driver = "docker"

      logs {
        max_files     = 2
        max_file_size = 10
      }

      env {
        NOMAD_ADDR = "http://${NOMAD_IP_http}:4646"
      }

      config {
        image = "ghcr.io/jumppad-labs/connector:v0.2.1"

        ports   = ["http", "grpc"]
        command = "/connector"
        args = [
          "run",
		      "--grpc-bind=:60000",
		      "--http-bind=:60001",
          "--log-level=%s",
          "--root-cert-path=local/certs/ca.cert",
          "--server-cert-path=local/certs/server.cert",
          "--server-key-path=local/certs/server.key",
          "--integration=nomad",
        ]
      }

      resources {
				# Use a single CPU to exhaust placement on the server node
        cpu    = 500
      }
    }
  }
}
`

const dataDir = `
data_dir = "/var/lib/nomad"
`

const serverConfig = `
server {
  enabled = true
  bootstrap_expect = 1
}

client {
	enabled = true
	meta {
		node_type = "server"
	}
	%s
}

plugin "raw_exec" {
  config {
		enabled = true
  }
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
