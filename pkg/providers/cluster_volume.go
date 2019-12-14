package providers

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/volume"
)

// createVolume creates a Docker volume for a cluster
// returns the volume name and an error if unsuccessful
func (c *Cluster) createVolume() (string, error) {

	name := fmt.Sprintf("%s.volume", c.config.Name)

	volumeCreateOptions := volume.VolumeCreateBody{
		Name:       name,
		Driver:     "local", //TODO: allow setting driver + opts
		DriverOpts: map[string]string{},
	}

	vol, err := c.client.VolumeCreate(context.Background(), volumeCreateOptions)
	if err != nil {
		return "", fmt.Errorf("failed to create image volume [%s] for cluster [%s]\n%+v", name, c.config.Name, err)
	}

	return vol.Name, nil
}

// deleteVolume deletes the Docker volume associated with  a cluster
func (c *Cluster) deleteVolume() {

}
