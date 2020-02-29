package clients

import (
	"context"
	"os"

	"github.com/hashicorp/go-getter"
	"golang.org/x/xerrors"
)

// Blueprints is an interface which defines interations for
// remote blueprints
type Blueprints interface {
	Get(uri, dst string) error
}

// BlueprintsImpl is a concrete implementation of the Blueprints interface
type BlueprintsImpl struct{}

// Get attempts to retrieve a blueprint
// from a remote location and stores it at the destination
// returns error on failure
func (bp *BlueprintsImpl) Get(uri, dst string) error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// if the argument is a url fetch it first
	c := &getter.Client{
		Ctx:     context.Background(),
		Src:     uri,
		Dst:     dst,
		Pwd:     pwd,
		Mode:    getter.ClientModeAny,
		Options: []getter.ClientOption{},
	}

	err = c.Get()
	if err != nil {
		return xerrors.Errorf("unable to fetch blueprint from %s: %w", uri, err)
	}

	return nil
}
