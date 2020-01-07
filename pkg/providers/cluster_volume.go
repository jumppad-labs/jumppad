package providers

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/volume"
)

// createVolume creates a Docker volume for a cluster
// returns the volume name and an error if unsuccessful
func (c *Cluster) createVolume() (string, error) {
	vn :=volumeName(c.config.Name)
	c.log.Debug("Create Volume", "ref", c.config.Name, "name", vn)

	volumeCreateOptions := volume.VolumeCreateBody{
		Name:       vn,
		Driver:     "local", //TODO: allow setting driver + opts
		DriverOpts: map[string]string{},
	}

	vol, err := c.client.VolumeCreate(context.Background(), volumeCreateOptions)
	if err != nil {
		return "", fmt.Errorf("failed to create image volume [%s] for cluster [%s]\n%+v", vn, c.config.Name, err)
	}

	return vol.Name, nil
}

// deleteVolume deletes the Docker volume associated with  a cluster
func (c *Cluster) deleteVolume() error { 
	vn := volumeName(c.config.Name)
	c.log.Debug("Deleting Volume", "ref", c.config.Name, "name", vn)

	return c.client.VolumeRemove(context.Background(), vn, true)
}

func volumeName(clusterName string) string {
	return fmt.Sprintf("%s.volume", clusterName)
}
