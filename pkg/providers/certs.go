package providers

import (
	"fmt"
	"os"
	"path"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/connector/crypto"
	"github.com/shipyard-run/shipyard/pkg/config"
	"golang.org/x/xerrors"
)

type CertificateCA struct {
	config *config.CertificateCA
	log    hclog.Logger
}

type CertificateLeaf struct {
	config *config.CertificateLeaf
	log    hclog.Logger
}

func NewCertificateCA(co *config.CertificateCA, l hclog.Logger) *CertificateCA {
	return &CertificateCA{co, l}
}

func NewCertificateLeaf(co *config.CertificateLeaf, l hclog.Logger) *CertificateLeaf {
	return &CertificateLeaf{co, l}
}

func (c *CertificateCA) Create() error {
	c.log.Info("Creating CA Certificate", "ref", c.config.Name)

	k, err := crypto.GenerateKeyPair()
	if err != nil {
		return err
	}

	ca, err := crypto.GenerateCA(k.Private)
	if err != nil {
		return err
	}

	err = k.Private.WriteFile(path.Join(c.config.Output, fmt.Sprintf("%s.key", c.config.Name)))
	if err != nil {
		return err
	}

	err = ca.WriteFile(path.Join(c.config.Output, fmt.Sprintf("%s.cert", c.config.Name)))
	if err != nil {
		return err
	}

	return nil
}

func (c *CertificateCA) Destroy() error {
	c.log.Info("Destroy CA Certificate", "ref", c.config.Name)

	err := os.Remove(path.Join(c.config.Output, fmt.Sprintf("%s.key", c.config.Name)))
	if err != nil {
		return err
	}

	err = os.Remove(path.Join(c.config.Output, fmt.Sprintf("%s.cert", c.config.Name)))
	if err != nil {
		return err
	}

	return nil
}

func (c *CertificateCA) Lookup() ([]string, error) {
	return nil, nil
}

func (c *CertificateLeaf) Create() error {
	c.log.Info("Creating Leaf Certificate", "ref", c.config.Name)

	// load the root key
	ca := &crypto.X509{}
	err := ca.ReadFile(c.config.CACert)
	if err != nil {
		return xerrors.Errorf("Unable to read root certificate %s: %w", c.config.CACert, err)
	}

	rk := crypto.NewKeyPair()
	err = rk.Private.ReadFile(c.config.CAKey)
	if err != nil {
		return xerrors.Errorf("Unable to read root key %s: %w", c.config.CAKey, err)
	}

	k, err := crypto.GenerateKeyPair()
	if err != nil {
		return err
	}

	err = k.Private.WriteFile(path.Join(c.config.Output, fmt.Sprintf("%s.key", c.config.Name)))
	if err != nil {
		return err
	}

	lc, err := crypto.GenerateLeaf(c.config.IPAddresses, c.config.DNSNames, ca, rk.Private, k.Private)
	if err != nil {
		return err
	}

	return lc.WriteFile(path.Join(c.config.Output, fmt.Sprintf("%s.cert", c.config.Name)))
}

func (c *CertificateLeaf) Destroy() error {
	c.log.Info("Destroy Leaf Certificate", "ref", c.config.Name)

	err := os.Remove(path.Join(c.config.Output, fmt.Sprintf("%s.key", c.config.Name)))
	if err != nil {
		c.log.Info("Unable to remove key", "ref", c.config.Name, "error", err)
	}

	err = os.Remove(path.Join(c.config.Output, fmt.Sprintf("%s.cert", c.config.Name)))
	if err != nil {
		c.log.Info("Unable to remove cert", "ref", c.config.Name, "error", err)
	}

	return nil
}

func (c *CertificateLeaf) Lookup() ([]string, error) {
	return nil, nil
}
