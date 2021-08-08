package providers

import (
	"fmt"
	"path/filepath"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/utils"
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
	ids, err := c.client.FindContainerIDs(c.config.Name, config.TypeContainer)
	if err != nil {
		return err
	}

	id := ""

	if ids == nil || len(ids) == 0 {
		var err error
		id, err = c.createImageCache()
		if err != nil {
			return err
		}
	} else {
		c.log.Debug("ImageCache already exists, not recreating")
		id = ids[0]
	}

	// remove all networks first
	// we should probably do a proper comparison
	c.detachFromNetworks(id)

	// connect to networks
	for _, n := range c.config.DependsOn {
		c.log.Debug("Connecting cache to network", "name", n)
		target, err := c.config.FindDependentResource(n)
		if err != nil {
			// ignore this network
			c.log.Warn("Unable to atttach cache to network, network does not exist", "name", n)
			continue
		}

		err = c.client.AttachNetwork(target.Info().Name, id, nil, "")
		if err != nil {
			return fmt.Errorf("Unable to attach cache to network: %s", err)
		}

		c.config.Networks = append(c.config.Networks, n)
	}

	return nil
}

func (c *ImageCache) createImageCache() (string, error) {
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

	return c.client.CreateContainer(cc)
}

func (c *ImageCache) Destroy() error {
	c.log.Info("Destroy ImageCache", "ref", c.config.Name)

	ids, err := c.client.FindContainerIDs(c.config.Name, c.config.Type)
	if err != nil {
		return err
	}

	// remove all networks that the container is attached to
	if len(ids) > 0 {
		for _, id := range ids {
			c.detachFromNetworks(id)
			c.client.RemoveContainer(id)
		}
	}

	return nil
}

func (c *ImageCache) detachFromNetworks(id string) {
	for _, n := range c.config.Networks {
		target, err := c.config.FindDependentResource(n)
		if err != nil {
			// ignore this resource
			continue
		}

		if target.Info().Type == config.TypeNetwork {
			c.log.Debug("Detaching container from network", "ref", c.config.Name, "id", id, "network", n)

			err := c.client.DetachNetwork(target.Info().Name, id)
			if err != nil {
				c.log.Error("Unable to detach network", "ref", c.config.Name, "network", target.Info().Name)
			}
		}
	}

	c.config.Networks = []string{}
}

func (c *ImageCache) Lookup() ([]string, error) {
	c.log.Info("Creating ImageCache", "ref", c.config.Name)
	return nil, nil
}
