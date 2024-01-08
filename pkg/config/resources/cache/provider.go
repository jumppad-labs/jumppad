package cache

import (
	"fmt"
	"path/filepath"
	"strings"

	ctypes "github.com/jumppad-labs/jumppad/pkg/config/resources/container"

	dtypes "github.com/docker/docker/api/types"
	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/container"
	"github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	sdk "github.com/jumppad-labs/plugin-sdk"
	"golang.org/x/xerrors"
)

const cacheImage = "ghcr.io/rpardini/docker-registry-proxy:0.6.4"
const defaultRegistries = "k8s.gcr.io gcr.io asia.gcr.io eu.gcr.io us.gcr.io quay.io ghcr.io docker.pkg.github.com"

type Provider struct {
	config *ImageCache
	client container.ContainerTasks
	log    logger.Logger
}

func (p *Provider) Init(cfg htypes.Resource, l sdk.Logger) error {
	c, ok := cfg.(*ImageCache)
	if !ok {
		return fmt.Errorf("unable to initialize ImageCache provider, resource is not of type ImageCache")
	}

	cli, err := clients.GenerateClients(l)
	if err != nil {
		return err
	}

	p.config = c
	p.client = cli.ContainerTasks
	p.log = l

	return nil
}

func (p *Provider) Create() error {
	p.log.Info("Creating ImageCache", "ref", p.config.ResourceID)

	// check the cache does not already exist
	ids, err := p.Lookup()
	if err != nil {
		return err
	}

	var registries []string
	var authRegistries []string

	for _, reg := range p.config.Registries {
		registries = append(registries, reg.Hostname)

		if reg.Auth != nil {
			host := reg.Hostname
			if reg.Auth.Hostname != "" {
				host = reg.Auth.Hostname
			}

			authRegistries = append(authRegistries, host+":::"+reg.Auth.Username+":::"+reg.Auth.Password)
		}
	}

	if len(ids) == 0 {
		_, err := p.createImageCache(registries, authRegistries)
		if err != nil {
			return err
		}
	}

	// get a list of dependent networks for the resource
	dependentNetworks := p.findDependentNetworks()

	// add the networks and return
	return p.reConfigureNetworks(dependentNetworks)
}

func (p *Provider) Destroy() error {
	p.log.Info("Destroy ImageCache", "ref", p.config.ResourceID)

	ids, err := p.Lookup()
	if err != nil {
		return err
	}

	if len(ids) > 0 {
		for _, id := range ids {
			err = p.client.RemoveContainer(id, true)
			if err != nil {
				p.log.Error(err.Error())
			}
		}
	}

	return nil
}

// Refresh is called whenever any Network resources are added or removed in Shipyard
// this is because we need to ensure that the cache is attached to all networks so that
// it can work with any clusters that may be on those networks.
func (p *Provider) Refresh() error {
	p.log.Debug("Refresh Image Cache", "ref", p.config.ResourceID)

	// get a list of dependent networks for the resource
	dependentNetworks := p.findDependentNetworks()

	return p.reConfigureNetworks(dependentNetworks)
}

func (p *Provider) Lookup() ([]string, error) {
	return p.client.FindContainerIDs(utils.FQDN(p.config.ResourceName, p.config.ResourceModule, p.config.ResourceType))
}

func (p *Provider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.ResourceID)

	return false, nil
}

func (p *Provider) createImageCache(registries []string, authRegistries []string) (string, error) {
	fqdn := utils.FQDN(p.config.ResourceName, p.config.ResourceModule, p.config.ResourceType)

	// Create the volume to store the cache
	// if this volume exists it will not be recreated
	volID, err := p.client.CreateVolume("images")
	if err != nil {
		return "", err
	}

	// copy the ca and key
	cert := filepath.Join(utils.CertsDir(""), "root.cert")
	key := filepath.Join(utils.CertsDir(""), "root.key")

	_, err = p.client.CopyFilesToVolume(volID, []string{cert, key}, "/ca", true)
	if err != nil {
		return "", xerrors.Errorf("unable to copy certificates for image cache: %w", err)
	}

	// pull the container image
	err = p.client.PullImage(types.Image{Name: cacheImage}, false)
	if err != nil {
		return "", err
	}

	// create the container
	cc := &types.Container{}
	cc.Name = fqdn
	cc.Image = &types.Image{Name: cacheImage}

	cc.Volumes = []types.Volume{
		{
			Source:      utils.FQDNVolumeName("images"),
			Destination: "/cache",
			Type:        "volume",
		},
	}

	cc.Environment = map[string]string{
		"CA_KEY_FILE":             "/cache/ca/root.key",
		"CA_CRT_FILE":             "/cache/ca/root.cert",
		"DEBUG":                   "false",
		"DEBUG_NGINX":             "false",
		"DEBUG_HUB":               "false",
		"DOCKER_MIRROR_CACHE":     "/cache/docker",
		"ENABLE_MANIFEST_CACHE":   "true",
		"REGISTRIES":              strings.Trim(defaultRegistries+" "+strings.Join(registries, " "), " "),
		"AUTH_REGISTRY_DELIMITER": ":::",
		"AUTH_REGISTRIES":         strings.Trim(strings.Join(authRegistries, " "), " "),
		"ALLOW_PUSH":              "true",
		"VERIFY_SSL":              "false",
	}

	// expose the docker proxy port on a random port num
	p1, err1 := utils.RandomAvailablePort(31000, 34000)
	p2, err2 := utils.RandomAvailablePort(31000, 34000)
	p3, err3 := utils.RandomAvailablePort(31000, 34000)

	if err1 != nil || err2 != nil || err3 != nil {
		return "", err
	}

	cc.Ports = []types.Port{
		{
			Local:    "3128",
			Host:     fmt.Sprintf("%d", p1),
			Protocol: "tcp",
		},
		{
			Local:    "8081",
			Host:     fmt.Sprintf("%d", p2),
			Protocol: "tcp",
		},
		{
			Local:    "8082",
			Host:     fmt.Sprintf("%d", p3),
			Protocol: "tcp",
		},
	}

	return p.client.CreateContainer(cc)
}

func (p *Provider) findDependentNetworks() []string {
	nets := []string{}

	for _, n := range p.config.DependsOn {
		if strings.HasSuffix(n, ".id") {
			// Ignore explicitly configured network dependencies
			continue
		}
		target, err := p.client.FindNetwork(n)
		if err != nil {
			// ignore this network
			p.log.Warn("A network ImageCache depends on does not exist", "name", n, "error", err)
			continue
		}

		nets = append(nets, target.Name)
	}

	return nets
}

// reConfigureNetworks updates the network attachments for the cache ensuring that it is
// attached to new networks that may have been added since the first run. And removed
// from any networks that may have been removed since the first run
func (p *Provider) reConfigureNetworks(dependentNetworks []string) error {
	currentNetworks := []string{}
	added := []string{}

	// get the container id
	ids, err := p.Lookup()
	if err != nil {
		return err
	}

	// cache not running
	if len(ids) == 0 {
		return nil
	}

	// get a list of the current networks the container is attached to
	info, err := p.client.ContainerInfo(ids[0])
	if err != nil {
		return xerrors.Errorf("unable to remove container from the default network: %w", err)
	}

	// flatten the docker object into a simple slice
	for k := range info.(dtypes.ContainerJSON).NetworkSettings.Networks {
		currentNetworks = append(currentNetworks, k)
	}

	// loop over the dependent networks and add the container to any that are missing
	for _, n := range dependentNetworks {
		// only add the network if it does not already exist
		if !contains(currentNetworks, n) {
			err = p.client.AttachNetwork(n, ids[0], nil, "")
			if err != nil {
				return fmt.Errorf("unable to attach cache to network: %s", err)
			}

			p.config.Networks = append(p.config.Networks, ctypes.NetworkAttachment{ID: n})
		}

		added = append(added, n)
	}

	// now remove any extra networks that are no longer required
	for _, n := range currentNetworks {
		if !contains(added, n) {
			p.log.Debug("Detaching container from network", "ref", p.config.ResourceID, "id", ids[0], "network", n)

			err := p.client.DetachNetwork(n, ids[0])
			if err != nil {
				p.log.Warn("Unable to detach network", "ref", p.config.ResourceID, "network", n)
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
