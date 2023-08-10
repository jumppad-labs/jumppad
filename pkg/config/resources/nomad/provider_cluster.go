package nomad

import (
	"bytes"
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
	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/connector"
	cclients "github.com/jumppad-labs/jumppad/pkg/clients/container"
	ctypes "github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/clients/nomad"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"golang.org/x/xerrors"
)

// NomadCluster defines a provider which can create Kubernetes clusters
type ClusterProvider struct {
	config      *NomadCluster
	client      cclients.ContainerTasks
	nomadClient nomad.Nomad
	connector   connector.Connector
	log         logger.Logger
}

var startTimeout = (300 * time.Second)

func (p *ClusterProvider) Init(cfg htypes.Resource, l logger.Logger) error {
	cli, err := clients.GenerateClients(l)
	if err != nil {
		return err
	}

	c, ok := cfg.(*NomadCluster)
	if !ok {
		return fmt.Errorf("unable to initialize NomadCluster provider, resource is not of type NomadCluster")
	}

	p.config = c
	p.client = cli.ContainerTasks
	p.nomadClient = cli.Nomad
	p.connector = cli.Connector
	p.log = l

	return nil
}

// Create implements interface method to create a cluster of the specified type
func (p *ClusterProvider) Create() error {
	return p.createNomad()
}

// Destroy implements interface method to destroy a cluster
func (p *ClusterProvider) Destroy() error {
	return p.destroyNomad()
}

// Lookup the a clusters current state
func (p *ClusterProvider) Lookup() ([]string, error) {
	ids := []string{}

	id, err := p.client.FindContainerIDs(p.config.ServerContainerName)
	if err != nil {
		return nil, err
	}

	ids = append(ids, id...)

	// find the clients
	for _, id := range p.config.ClientContainerName {
		id, err := p.client.FindContainerIDs(id)
		if err != nil {
			return nil, err
		}

		ids = append(ids, id...)
	}

	return ids, nil
}

// Refresh is called when `up` is run and the resource has been marked as created
// checks the nodes are healthy and replaces if needed.
func (p *ClusterProvider) Refresh() error {
	p.log.Debug("Refresh Nomad Cluster", "ref", p.config.ID)

	p.log.Debug("Checking health of server node", "ref", p.config.ID, "server", p.config.ServerContainerName)

	ids, _ := p.client.FindContainerIDs(p.config.ServerContainerName)
	if len(ids) == 1 {
		p.log.Debug("Server node exists", "ref", p.config.ID, "server", p.config.ServerContainerName, "id", ids[0])
	}

	// find any nodes that have crashed or have been deleted
	for _, cl := range p.config.ClientContainerName {
		p.log.Debug("Checking health of client nodes", "ref", p.config.ID, "client", cl)

		ids, _ := p.client.FindContainerIDs(cl)
		if len(ids) == 1 {
			p.log.Debug("Client node healthy", "ref", p.config.ID, "client", cl, "id", ids[0])
		} else {
			p.log.Debug("Client node does not exist", "ref", p.config.ID, "client", cl)
			// recreate the node
			p.config.ClientContainerName = removeElement(p.config.ClientContainerName, cl)
		}
	}

	// Has the number of clients nodes changed and are we scaling down?
	if p.config.ClientNodes < len(p.config.ClientContainerName) {
		// calculate the number of nodes that should be removed
		removeCount := len(p.config.ClientContainerName) - p.config.ClientNodes
		p.log.Info("Scaling cluster down", "ref", p.config.ID, "current_scale", len(p.config.ClientContainerName), "new_scale", p.config.ClientNodes, "removing", removeCount)

		// add the nodes to the remove list
		nodesToRemove := p.config.ClientContainerName[:removeCount]

		wg := sync.WaitGroup{}
		wg.Add(len(nodesToRemove))

		for _, n := range nodesToRemove {
			p.log.Debug("Removing node", "ref", p.config.ID, "client", n)

			go func(name string) {
				err := p.destroyNode(name)
				wg.Done()

				if err != nil {
					p.log.Error("Unable to remove node", "ref", p.config.ID, "client", name)
				}
			}(n)

			p.config.ClientContainerName = removeElement(p.config.ClientContainerName, n)
		}

		wg.Wait()

		p.nomadClient.SetConfig(fmt.Sprintf("http://%s", p.config.ExternalIP), p.config.APIPort, p.config.ClientNodes+1)
		err := p.nomadClient.HealthCheckAPI(startTimeout)
		if err != nil {
			return err
		}

		return nil
	}

	// do we need to scale the cluster up
	if p.config.ClientNodes > len(p.config.ClientContainerName) {
		// need to scale up
		p.log.Info("Scaling cluster up", "ref", p.config.ID, "current_scale", len(p.config.ClientContainerName), "new_scale", p.config.ClientNodes)

		for i := len(p.config.ClientContainerName); i < p.config.ClientNodes; i++ {
			id := utils.FQDN(fmt.Sprintf("%s.client.%s", randomID(), p.config.Name), p.config.Module, p.config.Type)

			p.log.Debug("Create client node", "ref", p.config.ID, "client", id)

			fqdn, _, err := p.createClientNode(randomID(), p.config.Image.Name, utils.ImageVolumeName, p.config.ServerContainerName)
			if err != nil {
				return fmt.Errorf(`unable to recreate client node "%s", %s`, id, err)
			}

			p.config.ClientContainerName = append(p.config.ClientContainerName, fqdn)

			p.log.Debug("Successfully created client node", "ref", p.config.ID, "client", fqdn)
		}

		p.nomadClient.SetConfig(fmt.Sprintf("http://%s", p.config.ExternalIP), p.config.APIPort, p.config.ClientNodes+1)
		err := p.nomadClient.HealthCheckAPI(startTimeout)
		if err != nil {
			return err
		}
	}

	// do we need to re-import any images?
	ci, err := p.getChangedImages()
	if err != nil {
		return err
	}

	if len(ci) > 0 {
		p.log.Info("Copied images changed, pushing new copy to the cluster", "ref", p.config.ID)
		err := p.ImportLocalDockerImages(ci, false)
		if err != nil {
			return err
		}
	}

	return nil
}

// PruneBuildImages removes any images
func (p *ClusterProvider) pruneBuildImages() error {
	ids, err := p.Lookup()
	if err != nil {
		return err
	}

	command := `docker rmi $(for IMAGE in $(docker images --filter reference="jumppad.dev/localcache/*" --format '{{.Repository}}' | sort | uniq); do docker images --filter reference=$IMAGE -q | awk '{if(NR>2)print}'; done)`

	output := bytes.NewBufferString("")

	for _, id := range ids {
		p.log.Debug("Prune build images from nomad node", "id", id)

		_, _ = p.client.ExecuteCommand(id, []string{"sh", "-c", command}, nil, "", "", "", 30, output)
		p.log.Debug("output", "result", output.String())
	}

	return nil
}

func (p *ClusterProvider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.ID)

	// check to see if the any of the copied images have changed
	i, err := p.getChangedImages()
	if err != nil {
		return false, err
	}

	if len(i) > 0 {
		return true, nil
	}

	return false, nil
}

func (p *ClusterProvider) createNomad() error {
	p.log.Info("Creating Cluster", "ref", p.config.ID)

	// check the client nodes do not already exist
	for i := 0; i < p.config.ClientNodes; i++ {
		ids, err := p.client.FindContainerIDs(utils.FQDN(fmt.Sprintf("%d.client.%s", i+1, p.config.Name), p.config.Module, p.config.Type))
		if len(ids) > 0 {
			return fmt.Errorf("client already exists")
		}

		if err != nil {
			return xerrors.Errorf("unable to lookup cluster id: %w", err)
		}
	}

	// check the server does not already exist
	ids, err := p.client.FindContainerIDs(utils.FQDN(fmt.Sprintf("server.%s", p.config.Name), p.config.Module, p.config.Type))
	if len(ids) > 0 {
		return fmt.Errorf("cluster already exists")
	}

	if err != nil {
		return xerrors.Errorf("unable to lookup cluster id: %w", err)
	}

	// pull the container image
	err = p.client.PullImage(p.config.Image.ToClientImage(), false)
	if err != nil {
		return err
	}

	// create the volume for the cluster
	volID, err := p.client.CreateVolume(utils.ImageVolumeName)
	if err != nil {
		return err
	}

	isClient := true
	if p.config.ClientNodes > 0 {
		isClient = false
	}

	// set the API server port to a random number
	p.config.ConnectorPort = rand.Intn(utils.MaxRandomPort-utils.MinRandomPort) + utils.MinRandomPort
	p.config.ConfigDir = path.Join(utils.JumppadHome(), p.config.Name, "config")
	p.config.ExternalIP = utils.GetDockerIP()

	_, err = p.createServerNode(p.config.Image.ToClientImage(), volID, isClient)
	if err != nil {
		return err
	}

	name := fmt.Sprintf("server.%s", p.config.Name)
	p.config.ServerContainerName = utils.FQDN(name, p.config.Module, p.config.Type)

	cMutex := sync.Mutex{}
	clientFQDN := []string{}
	clWait := sync.WaitGroup{}
	clWait.Add(p.config.ClientNodes)

	var clientError error
	for i := 0; i < p.config.ClientNodes; i++ {
		// create client node asynchronously
		go func(id string, image, volID, name string) {
			fqdn, _, err := p.createClientNode(id, image, volID, name)
			if err != nil {
				clientError = err
			}

			cMutex.Lock()

			clientFQDN = append(clientFQDN, fqdn)
			cMutex.Unlock()

			clWait.Done()
		}(randomID(), p.config.Image.Name, volID, p.config.ServerContainerName)
	}

	clWait.Wait()

	// set the client ids
	p.config.ClientContainerName = clientFQDN

	if clientError != nil {
		return xerrors.Errorf("Unable to create client nodes: %w", clientError)
	}

	// if client nodes is 0 then the server acts as both client and server
	// in this instance set the health check to 1 node
	clientNodes := 1

	// otherwise use the number of specified client nodes
	if p.config.ClientNodes > 0 {
		clientNodes = p.config.ClientNodes + 1
	}

	// ensure all client nodes are up
	p.nomadClient.SetConfig(fmt.Sprintf("http://%s", p.config.ExternalIP), p.config.APIPort, clientNodes)
	err = p.nomadClient.HealthCheckAPI(startTimeout)
	if err != nil {
		return err
	}

	// import the images to the servers container d instance
	// importing images means that Nomad does not need to pull from a remote docker hub
	if len(p.config.CopyImages) > 0 {
		err := p.ImportLocalDockerImages(p.config.CopyImages.ToClientImages(), false)
		if err != nil {
			return fmt.Errorf("unable to copy images to cluster: %w", err)
		}
	}

	err = p.deployConnector()
	if err != nil {
		return fmt.Errorf("unable to deploy Connector: %s", err)
	}

	return nil
}

func (p *ClusterProvider) createServerNode(img ctypes.Image, volumeID string, isClient bool) (string, error) {
	// set the resources for CPU, if not a client set the resources low
	// so that we can only deploy the connector to the server
	cpu := ""
	if !isClient {
		cpu = "cpu_total_compute = 500"
	}

	// generate the server config
	sc := dataDir + "\n" + fmt.Sprintf(serverConfig, cpu)

	// write the nomad config to a file
	os.MkdirAll(p.config.ConfigDir, os.ModePerm)
	serverConfigPath := path.Join(p.config.ConfigDir, "server_config.hcl")
	ioutil.WriteFile(serverConfigPath, []byte(sc), os.ModePerm)

	// create the server
	// since the server is just a container create the container config and provider
	name := fmt.Sprintf("server.%s", p.config.Name)
	fqrn := utils.FQDN(name, p.config.Module, p.config.Type)

	cc := &ctypes.Container{
		Name: fqrn,
	}

	cc.Image = &img
	cc.Networks = p.config.Networks.ToClientNetworkAttachments()
	cc.Privileged = true // nomad must run Privileged as Docker needs to manipulate ip tables and stuff

	// Add Consul DNS
	//cc.DNS = []string{"127.0.0.1"}

	// set the volume mount for the images and the config
	cc.Volumes = []ctypes.Volume{
		{
			Source:      volumeID,
			Destination: "/cache",
			Type:        "volume",
		},
		{
			Source:      serverConfigPath,
			Destination: "/etc/nomad.d/config.hcl",
			Type:        "bind",
		},
	}

	// Add any user config if set
	if p.config.ServerConfig != "" {
		vol := ctypes.Volume{
			Source:      p.config.ServerConfig,
			Destination: "/etc/nomad.d/server_user_config.hcl",
			Type:        "bind",
		}

		cc.Volumes = append(cc.Volumes, vol)
	}

	// Add any user config if set
	if p.config.ClientConfig != "" {
		vol := ctypes.Volume{
			Source:      p.config.ClientConfig,
			Destination: "/etc/nomad.d/client_user_config.hcl",
			Type:        "bind",
		}

		cc.Volumes = append(cc.Volumes, vol)
	}

	// Add the custom consul config if set
	if p.config.ConsulConfig != "" {
		vol := ctypes.Volume{
			Source:      p.config.ConsulConfig,
			Destination: "/etc/consul.d/config/user_config.hcl",
			Type:        "bind",
		}

		cc.Volumes = append(cc.Volumes, vol)
	}

	// if there are any custom volumes to mount
	for _, v := range p.config.Volumes {
		cc.Volumes = append(cc.Volumes, v.ToClientVolume())
	}

	cc.Environment = p.config.Environment

	// expose the API server port
	cc.Ports = []ctypes.Port{
		{
			Local:    "4646",
			Host:     fmt.Sprintf("%d", p.config.APIPort),
			Protocol: "tcp",
		},
		{
			Local:    fmt.Sprintf("%d", p.config.ConnectorPort),
			Host:     fmt.Sprintf("%d", p.config.ConnectorPort),
			Protocol: "tcp",
		},
		{
			Local:    fmt.Sprintf("%d", p.config.ConnectorPort+1),
			Host:     fmt.Sprintf("%d", p.config.ConnectorPort+1),
			Protocol: "tcp",
		},
	}

	cc.Ports = append(cc.Ports, p.config.Ports.ToClientPorts()...)
	cc.PortRanges = append(cc.PortRanges, p.config.PortRanges.ToClientPortRanges()...)

	cc.Environment = map[string]string{}
	err := p.appendProxyEnv(cc)
	if err != nil {
		return "", err
	}

	id, err := p.client.CreateContainer(cc)
	if err != nil {
		return "", err
	}

	return id, nil
}

// createClient node creates a Nomad client node
// returns the fqdn, docker id, and an error if unsuccessful
func (p *ClusterProvider) createClientNode(id string, image, volumeID, serverID string) (string, string, error) {
	// generate the client config
	sc := dataDir + "\n" + fmt.Sprintf(clientConfig, serverID)

	// write the default config to a file
	clientConfigPath := path.Join(p.config.ConfigDir, "client_config.hcl")
	ioutil.WriteFile(clientConfigPath, []byte(sc), os.ModePerm)

	// create the server
	// since the server is just a container create the container config and provider
	name := fmt.Sprintf("%s.client.%s", id, p.config.Name)
	fqrn := utils.FQDN(name, p.config.Module, p.config.Type)
	cc := &ctypes.Container{
		Name: fqrn,
	}

	cc.Image = &ctypes.Image{Name: image}
	cc.Networks = p.config.Networks.ToClientNetworkAttachments()
	cc.Privileged = true // nomad must run Privileged as Docker needs to manipulate ip tables and stuff

	//cc.DNS = []string{"127.0.0.1"}

	// set the volume mount for the images and the config
	cc.Volumes = []ctypes.Volume{
		{
			Source:      volumeID,
			Destination: "/cache",
			Type:        "volume",
		},
		{
			Source:      clientConfigPath,
			Destination: "/etc/nomad.d/config.hcl",
			Type:        "bind",
		},
	}

	// Add any user config if set
	if p.config.ClientConfig != "" {
		vol := ctypes.Volume{
			Source:      p.config.ClientConfig,
			Destination: "/etc/nomad.d/user_config.hcl",
			Type:        "bind",
		}

		cc.Volumes = append(cc.Volumes, vol)
	}

	// Add the custom consul config if set
	if p.config.ConsulConfig != "" {
		vol := ctypes.Volume{
			Source:      p.config.ConsulConfig,
			Destination: "/etc/consul.d/config/user_config.hcl",
			Type:        "bind",
		}

		cc.Volumes = append(cc.Volumes, vol)
	}

	// if there are any custom volumes to mount
	cc.Volumes = append(cc.Volumes, p.config.Volumes.ToClientVolumes()...)

	cc.Environment = p.config.Environment

	cc.Environment = map[string]string{}
	err := p.appendProxyEnv(cc)
	if err != nil {
		return "", "", err
	}

	cid, err := p.client.CreateContainer(cc)
	return fqrn, cid, err
}

func (p *ClusterProvider) appendProxyEnv(cc *ctypes.Container) error {
	// load the CA from a file
	ca, err := ioutil.ReadFile(filepath.Join(utils.CertsDir(""), "/root.cert"))
	if err != nil {
		return fmt.Errorf("unable to read root CA for proxy: %s", err)
	}

	// add the netmask from the network to the proxy bypass
	networkSubmasks := []string{}
	for _, n := range p.config.Networks {
		net, err := p.client.FindNetwork(n.ID)
		if err != nil {
			return fmt.Errorf("network not found: %w", err)
		}

		networkSubmasks = append(networkSubmasks, net.Subnet)
	}

	proxyBypass := utils.ProxyBypass + "," + strings.Join(networkSubmasks, ",")

	cc.Environment["HTTP_PROXY"] = utils.HTTPProxyAddress()
	cc.Environment["HTTPS_PROXY"] = utils.HTTPSProxyAddress()
	cc.Environment["NO_PROXY"] = proxyBypass
	cc.Environment["PROXY_CA"] = string(ca)

	return nil
}

func (p *ClusterProvider) deployConnector() error {
	p.log.Debug("Deploying connector", "ref", p.config.ID)

	// generate the certificates
	// generate the certificates for the service
	cb, err := p.connector.GetLocalCertBundle(utils.CertsDir(""))
	if err != nil {
		return fmt.Errorf("unable to fetch root certificates for ingress: %s", err)
	}

	// generate the leaf certificates ensuring that we add
	// the ip address for the docker hosts as this might not be local
	lf, err := p.connector.GenerateLeafCert(
		cb.RootKeyPath,
		cb.RootCertPath,
		[]string{
			"connector",
			fmt.Sprintf("%s:%d", utils.GetDockerIP(), p.config.ConnectorPort),
		},
		[]string{utils.GetDockerIP()},
		utils.CertsDir(p.config.ID),
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
		p.config.ConnectorPort,
		p.config.ConnectorPort+1,
		string(cert),
		string(key),
		string(ca),
		ll,
	)

	connectorDeployment := filepath.Join(dir, "connector.nomad")
	ioutil.WriteFile(connectorDeployment, []byte(config), os.ModePerm)

	// deploy the file
	err = p.nomadClient.Create([]string{connectorDeployment})
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

		ok, lastError = p.nomadClient.JobRunning("connector")
		if err != nil {
			lastError = fmt.Errorf("unable to check Connector deployment health: %s", err)
			continue
		}

		if ok {
			break
		}

		lastError = fmt.Errorf("connector not healthy")
	}

	return lastError
}

// ImportLocalDockerImages fetches Docker images stored on the local client and imports them into the cluster
func (p *ClusterProvider) ImportLocalDockerImages(images []ctypes.Image, force bool) error {
	ids, err := p.Lookup()
	if err != nil {
		return err
	}

	imgs := []string{}
	for _, i := range images {
		// ignore when the name is empty
		if i.Name == "" {
			continue
		}

		err := p.client.PullImage(i, false)
		if err != nil {
			return err
		}

		imgs = append(imgs, i.Name)
	}

	// import to volume
	vn := utils.FQDNVolumeName(utils.ImageVolumeName)
	imagesFile, err := p.client.CopyLocalDockerImagesToVolume(imgs, vn, force)
	if err != nil {
		return err
	}

	clWait := sync.WaitGroup{}
	clWait.Add(len(ids))

	for _, id := range ids {
		go func(ref, id string, images []string) {
			p.log.Debug("Importing docker images", "ref", p.config.ID, "id", id)
			// execute the command to import the image
			// write any command output to the logger
			for _, i := range images {
				_, err = p.client.ExecuteCommand(id, []string{"docker", "load", "-i", i}, nil, "/", "", "", 300, p.log.StandardWriter())
				if err != nil {
					p.log.Error("Unable to import docker images", "error", err)
				}
			}
			clWait.Done()
		}(p.config.ID, id, imagesFile)
	}

	// wait until all images have been imported
	clWait.Wait()

	// prune the build images
	p.pruneBuildImages()

	// update the config with the image ids
	p.updateCopyImageIDs()

	return nil
}

func (p *ClusterProvider) destroyNomad() error {
	p.log.Info("Destroy Nomad Cluster", "ref", p.config.ID)

	// destroy the clients
	wg := sync.WaitGroup{}
	wg.Add(len(p.config.ClientContainerName))

	for _, cl := range p.config.ClientContainerName {
		go func(name string) {
			defer wg.Done()

			err := p.destroyNode(name)
			if err != nil {
				p.log.Error("unable to remove cluster client", "client", name)
			}
		}(cl)
	}

	wg.Wait()

	// destroy the server
	err := p.destroyNode(p.config.ServerContainerName)
	if err != nil {
		return err
	}

	// remove the config
	os.RemoveAll(p.config.ConfigDir)

	return nil
}

func (p *ClusterProvider) destroyNode(id string) error {
	// FindContainerIDs works on absolute addresses, we need to append the server
	ids, _ := p.client.FindContainerIDs(id)
	if len(ids) == 0 {
		// nothing to do
		return nil
	}

	for _, i := range ids {
		// remove from the networks
		for _, n := range p.config.Networks {
			p.log.Debug("Detaching container from network", "ref", p.config.ID, "id", i, "network", n.Name)
			err := p.client.DetachNetwork(n.Name, i)
			if err != nil {
				p.log.Error("Unable to detach network", "ref", p.config.ID, "network", n.Name, "error", err)
			}
		}

		err := p.client.RemoveContainer(i, false)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *ClusterProvider) getChangedImages() ([]ctypes.Image, error) {
	changed := []ctypes.Image{}

	for _, i := range p.config.CopyImages {
		// has the image id changed
		id, err := p.client.FindImageInLocalRegistry(i.ToClientImage())
		if err != nil {
			p.log.Error("Unable to lookup image in local registry", "ref", p.config.ID, "error", err)
			return nil, err
		}

		// check that the current registry id for the image is the same
		// as the image that was used to create this container
		if id != i.ID {
			p.log.Debug("Container image changed, needs refresh", "ref", p.config.Name, "image", i.Name)
			changed = append(changed, i.ToClientImage())
		}
	}

	return changed, nil
}

// updates the ids for images that are copied to the container
// we store the image id in addition to the name so we can
// detect when it has changed
func (p *ClusterProvider) updateCopyImageIDs() error {
	for n, i := range p.config.CopyImages {
		id, err := p.client.FindImageInLocalRegistry(i.ToClientImage())
		if err != nil {
			return err
		}

		p.config.CopyImages[n].ID = id
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
