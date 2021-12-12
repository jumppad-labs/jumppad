package clients

import (
	"context"

	"github.com/containers/podman/v3/pkg/bindings"
)

// NewPodman creates a new Podman client
func NewPodman() (context.Context, error) {
	conn, err := bindings.NewConnection(context.Background(), "unix:///run/podman/podman.sock")
	if err != nil {
		return nil, err
	}

	return conn, nil
}
