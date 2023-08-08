package random

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/hashicorp/go-uuid"
	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
)

// RandomUUID is a provider for generating random UUIDs
type RandomUUIDProvider struct {
	config *RandomUUID
	log    logger.Logger
}

func (p *RandomUUIDProvider) Init(cfg htypes.Resource, l logger.Logger) error {
	c, ok := cfg.(*RandomUUID)
	if !ok {
		return fmt.Errorf("unable to initialize RandomUUID provider, resource is not of type RandomUUID")
	}

	p.config = c
	p.log = l

	return nil
}

func (p *RandomUUIDProvider) Create() error {
	result, err := uuid.GenerateUUID()
	if err != nil {
		return err
	}

	p.config.Value = result

	return nil
}

func (p *RandomUUIDProvider) Destroy() error {
	return nil
}

func (p *RandomUUIDProvider) Lookup() ([]string, error) {
	return nil, nil
}

func (p *RandomUUIDProvider) Refresh() error {
	return nil
}

func (p *RandomUUIDProvider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.ID)

	return false, nil
}

func generateRandomBytes(charSet *string, length int64) ([]byte, error) {
	bytes := make([]byte, length)
	if len(*charSet) == 0 {
		return bytes, nil
	}

	setLen := big.NewInt(int64(len(*charSet)))
	for i := range bytes {
		idx, err := rand.Int(rand.Reader, setLen)
		if err != nil {
			return nil, err
		}
		bytes[i] = (*charSet)[idx.Int64()]
	}
	return bytes, nil
}
