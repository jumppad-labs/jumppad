package providers

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/connector/crypto"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/pkg/errors"
	"github.com/sethvargo/go-retry"
	"golang.org/x/crypto/ssh"
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
	pubsshFile := path.Join(directory, fmt.Sprintf("%s.ssh", c.config.Name))
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

	// output the public ssh key
	ssh, err := publicPEMtoOpenSSH(k.Public.PEMBlock())
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(pubsshFile, []byte(ssh), os.ModePerm)
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

	c.config.PublicKeyPEM = &resources.File{
		Path:      pubkeyFile,
		Directory: directory,
		Filename:  fmt.Sprintf("%s.pub", c.config.Name),
		Contents:  k.Public.String(),
	}

	c.config.PublicKeySSH = &resources.File{
		Path:      pubsshFile,
		Directory: directory,
		Filename:  fmt.Sprintf("%s.ssh", c.config.Name),
		Contents:  ssh,
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
	pubsshFile := path.Join(directory, fmt.Sprintf("%s-leaf.ssh", c.config.Name))
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

		// output the public ssh key
		ssh, err := publicPEMtoOpenSSH(k.Public.PEMBlock())
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(pubsshFile, []byte(ssh), os.ModePerm)
		if err != nil {
			return err
		}

		// set the outputs
		c.config.PublicKeySSH = &resources.File{
			Path:      pubsshFile,
			Directory: directory,
			Filename:  fmt.Sprintf("%s-leaf.ssh", c.config.Name),
			Contents:  ssh,
		}

		c.config.PublicKeyPEM = &resources.File{
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

// thanks to https://gist.github.com/sriramsa/68d150ad50db4828f139e60a0efbde5a
func publicPEMtoOpenSSH(pemBytes []byte) (string, error) {
	// Decode and get the first block in the PEM file.
	// In our case it should be the Public key block.
	pemBlock, rest := pem.Decode(pemBytes)
	if pemBlock == nil {
		return "", errors.New("invalid PEM public key passed, pem.Decode() did not find a public key")
	}
	if len(rest) > 0 {
		return "", errors.New("PEM block contains more than just public key")
	}

	// Confirm we got the PUBLIC KEY block type
	if pemBlock.Type != "RSA PUBLIC KEY" {
		return "", errors.Errorf("ssh: unsupported key type %q", pemBlock.Type)
	}

	// Convert to rsa
	rsaPubKey, err := x509.ParsePKCS1PublicKey(pemBlock.Bytes)
	if err != nil {
		return "", errors.Wrap(err, "x509.parse pki public key")
	}

	// Generate the ssh public key
	pub, err := ssh.NewPublicKey(rsaPubKey)
	if err != nil {
		return "", errors.Wrap(err, "new ssh public key from pem converted to rsa")
	}

	// Encode to store to file
	sshPubKey := base64.StdEncoding.EncodeToString(pub.Marshal())

	return sshPubKey, nil
}
