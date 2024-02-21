package random

import (
	"crypto/rand"
	"fmt"
	"sort"

	htypes "github.com/jumppad-labs/hclconfig/types"
	sdk "github.com/jumppad-labs/plugin-sdk"
)

// RandomPassword is a provider for generating random passwords
type RandomPasswordProvider struct {
	config *RandomPassword
	log    sdk.Logger
}

func (p *RandomPasswordProvider) Init(cfg htypes.Resource, l sdk.Logger) error {
	c, ok := cfg.(*RandomPassword)
	if !ok {
		return fmt.Errorf("unable to initialize RandomPassword provider, resource is not of type RandomPassword")
	}

	p.config = c
	p.log = l

	return nil
}

func (p *RandomPasswordProvider) Create() error {
	const numChars = "0123456789"
	const lowerChars = "abcdefghijklmnopqrstuvwxyz"
	const upperChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var specialChars = "!@#$%&*()-_=+[]{}<>:?"
	var result []byte

	if p.config.OverrideSpecial != "" {
		specialChars = p.config.OverrideSpecial
	}

	var chars = ""
	if *p.config.Upper {
		chars += upperChars
	}

	if *p.config.Lower {
		chars += lowerChars
	}

	if *p.config.Numeric {
		chars += numChars
	}

	if *p.config.Special {
		chars += specialChars
	}

	minMapping := map[string]int64{
		numChars:     p.config.MinNumeric,
		lowerChars:   p.config.MinLower,
		upperChars:   p.config.MinUpper,
		specialChars: p.config.MinSpecial,
	}

	result = make([]byte, 0, p.config.Length)

	for k, v := range minMapping {
		s, err := generateRandomBytes(&k, v)
		if err != nil {
			return err
		}
		result = append(result, s...)
	}

	s, err := generateRandomBytes(&chars, p.config.Length-int64(len(result)))
	if err != nil {
		return err
	}

	result = append(result, s...)

	order := make([]byte, len(result))
	if _, err := rand.Read(order); err != nil {
		return err
	}

	sort.Slice(result, func(i, j int) bool {
		return order[i] < order[j]
	})

	p.config.Value = string(result)

	return nil
}

func (p *RandomPasswordProvider) Destroy() error {
	return nil
}

func (p *RandomPasswordProvider) Lookup() ([]string, error) {
	return nil, nil
}

func (p *RandomPasswordProvider) Refresh() error {
	return nil
}

func (p *RandomPasswordProvider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.Meta.ID)

	return false, nil
}
