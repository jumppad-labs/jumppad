package providers

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

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

	directory := strings.Replace(c.config.Module, ".", "_", -1)
	directory = path.Join(c.config.Output, directory)
	os.MkdirAll(directory, os.ModePerm)

	keyFile := path.Join(directory, fmt.Sprintf("%s-leaf.key", c.config.Name))
	pubkeyFile := path.Join(directory, fmt.Sprintf("%s-leaf.pub", c.config.Name))
	pubsshFile := path.Join(directory, fmt.Sprintf("%s-leaf.ssh", c.config.Name))
	certFile := path.Join(directory, fmt.Sprintf("%s-leaf.cert", c.config.Name))

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

	lc, err := crypto.GenerateLeaf(c.config.Name, c.config.IPAddresses, c.config.DNSNames, ca, rk.Private, k.Private)
	if err != nil {
		return err
	}

	// output the public ssh key
	ssh, err := publicPEMtoOpenSSH(k.Public.PEMBlock())
	if err != nil {
		return err
	}

	// Save the certificate
	err = lc.WriteFile(certFile)
	if err != nil {
		return err
	}

	// Save the keys
	err = k.Private.WriteFile(keyFile)
	if err != nil {
		return err
	}

	err = k.Public.WriteFile(pubkeyFile)
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

	return err
}

func (c *CertificateLeaf) Destroy() error {
	c.log.Info("Destroy Leaf Certificate", "ref", c.config.Name)

	directory := strings.Replace(c.config.Module, ".", "_", -1)
	directory = path.Join(c.config.Output, directory)
	os.MkdirAll(directory, os.ModePerm)

	return destroy(fmt.Sprintf("%s-%s", c.config.Name, "leaf"), directory, c.log)
}

func (c *CertificateLeaf) Lookup() ([]string, error) {
	return nil, nil
}

func (c *CertificateLeaf) Refresh() error {
	c.log.Info("Refresh Leaf Certificate", "ref", c.config.Name)

	return nil
}

func destroy(name, output string, log hclog.Logger) error {
	keyFile := path.Join(output, fmt.Sprintf("%s.key", name))
	pubkeyFile := path.Join(output, fmt.Sprintf("%s.pub", name))
	pubsshFile := path.Join(output, fmt.Sprintf("%s.ssh", name))
	certFile := path.Join(output, fmt.Sprintf("%s.cert", name))

	err := os.Remove(keyFile)
	if err != nil {
		log.Debug("Unable to remove private key", "ref", name, "error", err)
	}

	err = os.Remove(pubkeyFile)
	if err != nil {
		log.Debug("Unable to remove public key", "ref", name, "error", err)
	}

	err = os.Remove(pubsshFile)
	if err != nil {
		log.Debug("Unable to remove ssh key", "ref", name, "error", err)
	}

	err = os.Remove(certFile)
	if err != nil {
		log.Debug("Unable to remove certificate", "ref", name, "error", err)
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
