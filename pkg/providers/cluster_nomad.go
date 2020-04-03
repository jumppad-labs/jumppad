package providers

import (
	"fmt"
	"math/rand"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"golang.org/x/xerrors"
)

const nomadBaseImage = "shipyardrun/nomad"

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
	return c.client.FindContainerIDs(c.config.Name, c.config.Type)
}

func (c *NomadCluster) createNomad() error {
	c.log.Info("Creating Cluster", "ref", c.config.Name)

	// check the cluster does not already exist
	ids, err := c.client.FindContainerIDs(c.config.Name, c.config.Type)
	if len(ids) > 0 {
		return ErrorClusterExists
	}

	if err != nil {
		return xerrors.Errorf("Unable to lookup cluster id: %w", err)
	}

	// set the image
	image := fmt.Sprintf("%s:%s", nomadBaseImage, c.config.Version)

	// pull the container image
	err = c.client.PullImage(config.Image{Name: image}, false)
	if err != nil {
		return err
	}

	// create the volume for the cluster
	volID, err := c.client.CreateVolume(c.config.Name)
	if err != nil {
		return err
	}

	// create the server
	// since the server is just a container create the container config and provider
	cc := config.NewContainer(fmt.Sprintf("server.%s", c.config.Name))
	c.config.ResourceInfo.AddChild(cc)

	cc.Image = config.Image{Name: image}
	cc.Networks = c.config.Networks
	cc.Privileged = true // nomad must run Privileged as Docker needs to manipulate ip tables and stuff

	// set the volume mount for the images
	cc.Volumes = []config.Volume{
		config.Volume{
			Source:      volID,
			Destination: "/images",
			Type:        "volume",
		},
	}

	// if there are any custom volumes to mount
	for _, v := range c.config.Volumes {
		cc.Volumes = append(cc.Volumes, v)
	}

	// set the environment variables for the K3S_KUBECONFIG_OUTPUT and K3S_CLUSTER_SECRET
	cc.Environment = c.config.Environment

	// set the API server port to a random number 64000 - 65000
	apiPort := rand.Intn(1000) + 64000

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
		return err
	}

	// generate the config file
	nomadConfig := clients.NomadConfig{Location: fmt.Sprintf("http://localhost:%d", apiPort), NodeCount: 1}
	_, configPath := utils.CreateNomadConfigPath(c.config.Name)

	err = nomadConfig.Save(configPath)
	if err != nil {
		return xerrors.Errorf("Unable to generate Nomad config: %w", err)
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
		err := c.ImportLocalDockerImages(c.config.Name, id, c.config.Images)
		if err != nil {
			return xerrors.Errorf("Error importing Docker images: %w", err)
		}
	}

	return nil
}

// ImportLocalDockerImages fetches Docker images stored on the local client and imports them into the cluster
func (c *NomadCluster) ImportLocalDockerImages(name string, id string, images []config.Image) error {
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
	imageFile, err := c.client.CopyLocalDockerImageToVolume(imgs, vn)
	if err != nil {
		return err
	}

	// execute the command to import the image
	// write any command output to the logger
	err = c.client.ExecuteCommand(id, []string{"docker", "load", "-i", "/images/" + imageFile}, c.log.StandardWriter(&hclog.StandardLoggerOptions{ForceLevel: hclog.Debug}))
	if err != nil {
		return err
	}

	return nil
}

func (c *NomadCluster) destroyNomad() error {
	c.log.Info("Destroy Cluster", "ref", c.config.Name)

	ids, err := c.client.FindContainerIDs(c.config.Name, c.config.Type)
	if err != nil {
		return err
	}

	for _, i := range ids {
		// remove from the networks
		for _, n := range c.config.Networks {
			err := c.client.DetachNetwork(n.Name, i)
			if err != nil {
				return err
			}
		}

		err := c.client.RemoveContainer(i)
		if err != nil {
			return err
		}
	}

	// delete the volume
	return c.client.RemoveVolume(c.config.Name)
}
