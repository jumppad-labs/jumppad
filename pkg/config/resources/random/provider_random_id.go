package random

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"

	htypes "github.com/jumppad-labs/hclconfig/types"
	sdk "github.com/jumppad-labs/plugin-sdk"
)

var _ sdk.Provider = &RandomIDProvider{}

// RandomID is a provider for generating random IDs
type RandomIDProvider struct {
	config *RandomID
	log    sdk.Logger
}

func (p *RandomIDProvider) Init(cfg htypes.Resource, l sdk.Logger) error {
	c, ok := cfg.(*RandomID)
	if !ok {
		return fmt.Errorf("unable to initialize RandomID provider, resource is not of type RandomID")
	}

	p.config = c
	p.log = l

	return nil
}

func (p *RandomIDProvider) Create(ctx context.Context) error {
	byteLength := p.config.ByteLength
	bytes := make([]byte, byteLength)

	b, err := rand.Reader.Read(bytes)
	if int64(b) != byteLength {
		return fmt.Errorf("unable generate random bytes: %w", err)
	}
	if err != nil {
		return fmt.Errorf("unable generate random bytes: %w", err)
	}

	hex := hex.EncodeToString(bytes)

	bigInt := big.Int{}
	bigInt.SetBytes(bytes)
	dec := bigInt.String()

	p.config.Hex = hex
	p.config.Dec = dec

	return nil
}

func (p *RandomIDProvider) Destroy(ctx context.Context, force bool) error {
	return nil
}

func (p *RandomIDProvider) Lookup() ([]string, error) {
	return nil, nil
}

func (p *RandomIDProvider) Refresh(ctx context.Context) error {
	return nil
}

func (p *RandomIDProvider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.Meta.ID)

	return false, nil
}
