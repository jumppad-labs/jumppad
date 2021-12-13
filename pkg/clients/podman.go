package clients

import (
	"fmt"
	"os/user"

	"github.com/docker/docker/client"
)

// NewPodman creates a new Podman client
func NewPodman() (Docker, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}

	cli, err := client.NewClientWithOpts(
		client.WithHost(fmt.Sprintf(
			"unix:///var/run/user/%s/podman/podman.sock",
			u.Uid,
		)),
	)
	if err != nil {
		return nil, err
	}

	return cli, nil
}
