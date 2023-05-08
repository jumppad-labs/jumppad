package providers

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/connector/crypto"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/sethvargo/go-retry"
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

	directory := strings.Replace(c.config.Module, ".", "_", -1)
	directory = path.Join(c.config.Output, directory)
	os.MkdirAll(directory, os.ModePerm)

	keyFile := path.Join(directory, fmt.Sprintf("%s.key", c.config.Name))
	pubkeyFile := path.Join(directory, fmt.Sprintf("%s.pub", c.config.Name))
	certFile := path.Join(directory, fmt.Sprintf("%s.cert", c.config.Name))

	k, err := crypto.GenerateKeyPair()
	if err != nil {
		return err
	}

	ca, err := crypto.GenerateCA(c.config.Name, k.Private)
	if err != nil {
		return err
	}

	err = k.Private.WriteFile(keyFile)
	if err != nil {
		return err
	}

	err = k.Public.WriteFile(pubkeyFile)
	if err != nil {
		return err
	}

	err = ca.WriteFile(certFile)
	if err != nil {
		return err
	}

	// set the outputs
	c.config.Cert = &resources.File{
		Path:      certFile,
		Directory: directory,
		Filename:  fmt.Sprintf("%s.cert", c.config.Name),
		Contents:  ca.String(),
	}

	c.config.PrivateKey = &resources.File{
		Path:      keyFile,
		Directory: directory,
		Filename:  fmt.Sprintf("%s.key", c.config.Name),
		Contents:  k.Private.String(),
	}

	c.config.PublicKey = &resources.File{
		Path:      pubkeyFile,
		Directory: directory,
		Filename:  fmt.Sprintf("%s.pub", c.config.Name),
		Contents:  k.Public.String(),
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

func (c *CertificateCA) Refresh() error {
	c.log.Info("Refresh CA Certificate", "ref", c.config.Name)

	return nil
}

func (c *CertificateLeaf) Create() error {
	c.log.Info("Creating Leaf Certificate", "ref", c.config.Name)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	directory := strings.Replace(c.config.Module, ".", "_", -1)
	directory = path.Join(c.config.Output, directory)
	os.MkdirAll(directory, os.ModePerm)

	keyFile := path.Join(directory, fmt.Sprintf("%s-leaf.key", c.config.Name))
	pubkeyFile := path.Join(directory, fmt.Sprintf("%s-leaf.pub", c.config.Name))
	certFile := path.Join(directory, fmt.Sprintf("%s-leaf.cert", c.config.Name))

	err := retry.Constant(ctx, 1*time.Second, func(ctx context.Context) error {
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
		err = k.Private.WriteFile(keyFile)
		if err != nil {
			return err
		}

		err = k.Public.WriteFile(pubkeyFile)
		if err != nil {
			return err
		}

		lc, err := crypto.GenerateLeaf(c.config.Name, c.config.IPAddresses, c.config.DNSNames, ca, rk.Private, k.Private)
		if err != nil {
			return err
		}

		// set the outputs
		c.config.PublicKey = &resources.File{
			Path:      pubkeyFile,
			Directory: directory,
			Filename:  fmt.Sprintf("%s-leaf.pub", c.config.Name),
			Contents:  k.Public.String(),
		}

		c.config.Cert = &resources.File{
			Path:      certFile,
			Directory: directory,
			Filename:  fmt.Sprintf("%s-leaf.cert", c.config.Name),
			Contents:  lc.String(),
		}

		c.config.PrivateKey = &resources.File{
			Path:      keyFile,
			Directory: directory,
			Filename:  fmt.Sprintf("%s-leaf.key", c.config.Name),
			Contents:  k.Private.String(),
		}

		// Save the certificate
		return lc.WriteFile(certFile)
	})

	return err
}

func (c *CertificateLeaf) Destroy() error {
	c.log.Info("Destroy Leaf Certificate", "ref", c.config.Name)

	directory := strings.Replace(c.config.Module, ".", "_", -1)
	directory = path.Join(c.config.Output, directory)
	os.MkdirAll(directory, os.ModePerm)

	return destroy(c.config.Name, directory, c.log)
}

func (c *CertificateLeaf) Lookup() ([]string, error) {
	return nil, nil
}

func (c *CertificateLeaf) Refresh() error {
	c.log.Info("Refresh Leaf Certificate", "ref", c.config.Name)

	return nil
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
