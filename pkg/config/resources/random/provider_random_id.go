package random

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"

	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"golang.org/x/xerrors"
)

// RandomID is a provider for generating random IDs
type RandomIDProvider struct {
	config *RandomID
	log    logger.Logger
}

func (p *RandomIDProvider) Init(cfg htypes.Resource, l logger.Logger) error {
	c, ok := cfg.(*RandomID)
	if !ok {
		return fmt.Errorf("unable to initialize RandomID provider, resource is not of type RandomID")
	}

	p.config = c
	p.log = l

	return nil
}

func (p *RandomIDProvider) Create() error {
	byteLength := p.config.ByteLength
	bytes := make([]byte, byteLength)

	b, err := rand.Reader.Read(bytes)
	if int64(b) != byteLength {
		return xerrors.Errorf("Unable generate random bytes: %w", err)
	}
	if err != nil {
		return xerrors.Errorf("Unable generate random bytes: %w", err)
	}

	hex := hex.EncodeToString(bytes)

	bigInt := big.Int{}
	bigInt.SetBytes(bytes)
	dec := bigInt.String()

	p.config.Hex = hex
	p.config.Dec = dec

	return nil
}

func (p *RandomIDProvider) Destroy() error {
	return nil
}

func (p *RandomIDProvider) Lookup() ([]string, error) {
	return nil, nil
}

func (p *RandomIDProvider) Refresh() error {
	return nil
}

func (p *RandomIDProvider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.ID)

	return false, nil
}
