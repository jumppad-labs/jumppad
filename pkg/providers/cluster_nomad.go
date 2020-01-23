package providers

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/shipyard-run/shipyard/pkg/config"
)

const nomadBaseImage = "shipyardrun/nomad"

func (c *Cluster) createNomad() error {
	c.log.Info("Creating Cluster", "ref", c.config.Name)

	// check the cluster does not already exist
	id, _ := c.Lookup()
	if id != "" {
		return ErrorClusterExists
	}

	// set the image
	image := fmt.Sprintf("%s:%s", nomadBaseImage, c.config.Version)

	// create the volume for the cluster
	volID, err := c.createVolume()
	if err != nil {
		return err
	}

	// create the server
	// since the server is just a container create the container config and provider
	cc := &config.Container{}
	cc.Name = fmt.Sprintf("server.%s", c.config.Name)
	cc.Image = config.Image{Name: image}
	cc.NetworkRef = c.config.NetworkRef
	cc.Privileged = true // nomad must run Privileged as Docker needs to manipulate ip tables and stuff

	// set the volume mount for the images
	cc.Volumes = []config.Volume{
		config.Volume{
			Source:      volID,
			Destination: "/images",
			Type:        "volume",
		},
	}

	// set the environment variables for the K3S_KUBECONFIG_OUTPUT and K3S_CLUSTER_SECRET
	cc.Environment = c.config.Environment

	// set the API server port to a random number 64000 - 65000
	apiPort := rand.Intn(1000) + 64000

	// expose the API server port
	cc.Ports = []config.Port{
		config.Port{
			Local:    4646,
			Host:     apiPort,
			Protocol: "tcp",
		},
	}

	cp := NewContainer(cc, c.client, c.log.With("parent_ref", c.config.Name))
	err = cp.Create()
	if err != nil {
		return err
	}

	// get the id
	id, err = c.Lookup()
	if err != nil {
		return err
	}

	// import the images to the servers docker instance
	// importing images means that Nomad does not need to pull from a remote docker hub
	if c.config.Images != nil && len(c.config.Images) > 0 {
		//return c.ImportLocalDockerImages(c.config.Images)
	}

	// wait for nomad to start
	err = healthCheckHTTP(fmt.Sprintf("http://localhost:%d/v1/status/leader", apiPort), 60*time.Second, c.log)
	if err != nil {
		return err
	}

	return nil
}

func (c *Cluster) destroyNomad() error {
	return c.destroyK3s()
}
