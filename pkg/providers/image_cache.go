package providers

import (
	"fmt"
	"math/rand"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"golang.org/x/xerrors"
)

const cacheImage = "shipyardrun/docker-registry-proxy:0.6.3"

type ImageCache struct {
	config     *config.ImageCache
	client     clients.ContainerTasks
	httpClient clients.HTTP
	log        hclog.Logger
}

// NewContainer creates a new container with the given config and Docker client
func NewImageCache(co *config.ImageCache, cl clients.ContainerTasks, hc clients.HTTP, l hclog.Logger) *ImageCache {

	return &ImageCache{co, cl, hc, l}
}

func (c *ImageCache) Create() error {
	c.log.Info("Creating ImageCache", "ref", c.config.Name)

	// check the cache does not already exist
	ids, err := c.client.FindContainerIDs(c.config.Name, c.config.Type)
	if err != nil {
		return err
	}

	id := ""

	// get a list of dependent networks for the resource
	dependentNetworks := c.findDependentNetworks()

	if len(ids) == 0 {
		var err error
		id, err = c.createImageCache(dependentNetworks)
		if err != nil {
			return err
		}

		return nil
	}

	c.log.Debug("ImageCache already exists, not recreating")
	id = ids[0]

	// Create is called whenever any Network resources are added or removed in Shipyard
	// this is because we need to ensure that the cache is attached to all networks so that
	// it can work with any clusters that may be on those networks.
	return c.reConfigureNetworks(id, dependentNetworks)
}

func (c *ImageCache) createImageCache(networks []string) (string, error) {
	// Create the volume to store the cache
	// if this volume exists it will not be recreated
	volID, err := c.client.CreateVolume("images")
	if err != nil {
		return "", err
	}

	// copy the ca and key
	cert := filepath.Join(utils.CertsDir(""), "root.cert")
	key := filepath.Join(utils.CertsDir(""), "root.key")

	_, err = c.client.CopyFilesToVolume(volID, []string{cert, key}, "/ca", true)
	if err != nil {
		return "", fmt.Errorf("Unable to copy certificates for image cache: %s", err)
	}

	// pull the container image
	err = c.client.PullImage(config.Image{Name: cacheImage}, false)
	if err != nil {
		return "", err
	}

	// create the container
	cc := config.NewContainer(c.config.Name)
	cc.Type = c.config.Type
	cc.Image = &config.Image{Name: cacheImage}

	cc.Volumes = []config.Volume{
		config.Volume{
			Source:      utils.FQDNVolumeName("images"),
			Destination: "/cache",
			Type:        "volume",
		},
	}

	cc.EnvVar = map[string]string{
		"CA_KEY_FILE":           "/cache/ca/root.key",
		"CA_CRT_FILE":           "/cache/ca/root.cert",
		"DOCKER_MIRROR_CACHE":   "/cache/docker",
		"ENABLE_MANIFEST_CACHE": "true",
		"REGISTRIES":            "k8s.gcr.io gcr.io asia.gcr.io eu.gcr.io us.gcr.io quay.io ghcr.io docker.pkg.github.com",
		"ALLOW_PUSH":            "true",
	}

	// expose the docker proxy port on a random port num
	cc.Ports = []config.Port{
		config.Port{
			Local:    "3128",
			Host:     fmt.Sprintf("%d", rand.Intn(3000)+31000),
			Protocol: "tcp",
		},
	}

	// add the networks
	cc.Networks = []config.NetworkAttachment{}
	for _, n := range networks {
		cc.Networks = append(cc.Networks, config.NetworkAttachment{Name: fmt.Sprintf("%s.%s", config.TypeNetwork, n)})
	}

	c.config.ResourceInfo.AddChild(cc)

	return c.client.CreateContainer(cc)
}

func (c *ImageCache) Destroy() error {
	c.log.Info("Destroy ImageCache", "ref", c.config.Name)

	ids, err := c.client.FindContainerIDs(c.config.Name, c.config.Type)
	if err != nil {
		return err
	}

	if len(ids) > 0 {
		for _, id := range ids {
			c.client.RemoveContainer(id, true)
		}
	}

	return nil
}

func (c *ImageCache) findDependentNetworks() []string {
	nets := []string{}

	for _, n := range c.config.DependsOn {
		c.log.Debug("Connecting cache to network", "name", n)
		target, err := c.config.FindDependentResource(n)
		if err != nil {
			// ignore this network
			c.log.Warn("Unable to atttach cache to network, network does not exist", "name", n)
			continue
		}

		if target.Info().Type == config.TypeNetwork {
			nets = append(nets, target.Info().Name)
		}
	}

	return nets
}

// reConfigureNetworks updates the network attachments for the cache ensuring that it is
// attached to new networks that may have been added since the first run. And removed
// from any networks that may have been removed since the first run
func (c *ImageCache) reConfigureNetworks(id string, dependentNetworks []string) error {
	currentNetworks := []string{}
	added := []string{}

	// get a list of the current networks the container is attached to
	info, err := c.client.ContainerInfo(id)
	if err != nil {
		return xerrors.Errorf("Unable to remove container from the default network: %w", err)
	}

	// flattern the docker object into a simple slice
	for k, _ := range info.(types.ContainerJSON).NetworkSettings.Networks {
		currentNetworks = append(currentNetworks, k)
	}

	// loop over the dependent networks and add the container to any that are missing
	for _, n := range dependentNetworks {
		// only add the network if it does not already exist
		if !contains(currentNetworks, n) {
			err = c.client.AttachNetwork(n, id, nil, "")
			if err != nil {
				return fmt.Errorf("Unable to attach cache to network: %s", err)
			}

			c.config.Networks = append(c.config.Networks, n)
		}

		added = append(added, n)
	}

	// now remove any extra networks that are no longer required
	for _, n := range currentNetworks {
		if !contains(added, n) {
			c.log.Debug("Detaching container from network", "ref", c.config.Name, "id", id, "network", n)

			err := c.client.DetachNetwork(n, id)
			if err != nil {
				c.log.Warn("Unable to detach network", "ref", c.config.Name, "network", n)
			}
		}
	}

	return nil
}

func contains(strings []string, s string) bool {
	for _, in := range strings {
		if in == s {
			return true
		}
	}

	return false
}

func (c *ImageCache) Lookup() ([]string, error) {
	c.log.Info("Creating ImageCache", "ref", c.config.Name)
	return nil, nil
}
