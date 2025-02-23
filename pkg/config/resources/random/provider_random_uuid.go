package random

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/hashicorp/go-uuid"

	htypes "github.com/jumppad-labs/hclconfig/types"
	sdk "github.com/jumppad-labs/plugin-sdk"
)

var _ sdk.Provider = &RandomUUIDProvider{}

// RandomUUID is a provider for generating random UUIDs
type RandomUUIDProvider struct {
	config *RandomUUID
	log    sdk.Logger
}

func (p *RandomUUIDProvider) Init(cfg htypes.Resource, l sdk.Logger) error {
	c, ok := cfg.(*RandomUUID)
	if !ok {
		return fmt.Errorf("unable to initialize RandomUUID provider, resource is not of type RandomUUID")
	}

	p.config = c
	p.log = l

	return nil
}

func (p *RandomUUIDProvider) Create(ctx context.Context) error {
	result, err := uuid.GenerateUUID()
	if err != nil {
		return err
	}

	p.config.Value = result

	return nil
}

func (p *RandomUUIDProvider) Destroy(ctx context.Context, force bool) error {
	return nil
}

func (p *RandomUUIDProvider) Lookup() ([]string, error) {
	return nil, nil
}

func (p *RandomUUIDProvider) Refresh(ctx context.Context) error {
	return nil
}

func (p *RandomUUIDProvider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.Meta.ID)

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
