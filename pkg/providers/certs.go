package providers

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/sethvargo/go-retry"
	"github.com/shipyard-run/connector/crypto"
	"github.com/shipyard-run/shipyard/pkg/config/resources"
	"golang.org/x/xerrors"
)

type CertificateCA struct {
	config *resources.CertificateCA
	log    hclog.Logger
}

type CertificateLeaf struct {
	config *resources.CertificateLeaf
	log    hclog.Logger
}

func NewCertificateCA(co *resources.CertificateCA, l hclog.Logger) *CertificateCA {
	return &CertificateCA{co, l}
}

func NewCertificateLeaf(co *resources.CertificateLeaf, l hclog.Logger) *CertificateLeaf {
	return &CertificateLeaf{co, l}
}

func (c *CertificateCA) Create() error {
	c.log.Info("Creating CA Certificate", "ref", c.config.Name)

	k, err := crypto.GenerateKeyPair()
	if err != nil {
		return err
	}

	ca, err := crypto.GenerateCA(c.config.Name, k.Private)
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

	return destroy(c.config.Name, c.config.Output, c.log)
}

func (c *CertificateCA) Lookup() ([]string, error) {
	return nil, nil
}

func (c *CertificateLeaf) Create() error {
	c.log.Info("Creating Leaf Certificate", "ref", c.config.Name)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return retry.Constant(ctx, 1*time.Second, func(ctx context.Context) error {
		ca := &crypto.X509{}
		err := ca.ReadFile(c.config.CACert)
		if err != nil {
			return retry.RetryableError(xerrors.Errorf("Unable to read root certificate %s: %w", c.config.CACert, err))
		}

		rk := crypto.NewKeyPair()
		err = rk.Private.ReadFile(c.config.CAKey)
		if err != nil {
			return retry.RetryableError(xerrors.Errorf("Unable to read root key %s: %w", c.config.CAKey, err))
		}

		k, err := crypto.GenerateKeyPair()
		if err != nil {
			return err
		}

		// Save the key
		err = k.Private.WriteFile(path.Join(c.config.Output, fmt.Sprintf("%s.key", c.config.Name)))
		if err != nil {
			return err
		}

		lc, err := crypto.GenerateLeaf(c.config.Name, c.config.IPAddresses, c.config.DNSNames, ca, rk.Private, k.Private)
		if err != nil {
			return err
		}

		// Save the certificate
		return lc.WriteFile(path.Join(c.config.Output, fmt.Sprintf("%s.cert", c.config.Name)))
	}) // Load the root key
}

func (c *CertificateLeaf) Destroy() error {
	c.log.Info("Destroy Leaf Certificate", "ref", c.config.Name)

	return destroy(c.config.Name, c.config.Output, c.log)
}

func (c *CertificateLeaf) Lookup() ([]string, error) {
	return nil, nil
}

func destroy(name, output string, log hclog.Logger) error {
	kp, _ := filepath.Abs(path.Join(output, fmt.Sprintf("%s.key", name)))
	err := os.Remove(kp)
	if err != nil {
		log.Debug("Unable to remove key", "ref", name, "error", err)
	}

	cp, _ := filepath.Abs(path.Join(output, fmt.Sprintf("%s.cert", name)))
	err = os.Remove(cp)
	if err != nil {
		log.Debug("Unable to remove cert", "ref", name, "error", err)
	}

	return nil
}
