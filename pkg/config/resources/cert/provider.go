package cert

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/jumppad-labs/connector/crypto"
	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	sdk "github.com/jumppad-labs/plugin-sdk"
	"github.com/pkg/errors"
	"github.com/sethvargo/go-retry"
	"golang.org/x/crypto/ssh"
)

type CAProvider struct {
	config *CertificateCA
	log    sdk.Logger
}

type LeafProvider struct {
	config *CertificateLeaf
	log    sdk.Logger
}

func (p *CAProvider) Init(cfg htypes.Resource, l sdk.Logger) error {
	c, ok := cfg.(*CertificateCA)
	if !ok {
		return fmt.Errorf("unable to initialize CA provider, resource is not of type CertificateCA")
	}

	p.config = c
	p.log = l
	return nil
}

func (p *LeafProvider) Init(cfg htypes.Resource, l sdk.Logger) error {
	c, ok := cfg.(*CertificateLeaf)
	if !ok {
		return fmt.Errorf("unable to initialize Leaf provider, resource is not of type CertificateLeaf")
	}

	p.config = c
	p.log = l
	return nil
}

func (p *CAProvider) Create(ctx context.Context) error {
	if ctx.Err() != nil {
		p.log.Debug("Context cancelled, skipping CA", "ref", p.config.Meta.ID)
		return nil
	}

	p.log.Info("Creating CA Certificate", "ref", p.config.Meta.ID)

	directory := strings.Replace(p.config.Meta.Module, ".", "_", -1)
	directory = path.Join(p.config.Output, directory)
	os.MkdirAll(directory, os.ModePerm)

	keyFile := path.Join(directory, fmt.Sprintf("%s.key", p.config.Meta.Name))
	publicKeyFile := path.Join(directory, fmt.Sprintf("%s.pub", p.config.Meta.Name))
	publicSSHFile := path.Join(directory, fmt.Sprintf("%s.ssh", p.config.Meta.Name))
	certificateFile := path.Join(directory, fmt.Sprintf("%s.cert", p.config.Meta.Name))

	k, err := crypto.GenerateKeyPair()
	if err != nil {
		return err
	}

	ca, err := crypto.GenerateCA(p.config.Meta.Name, k.Private)
	if err != nil {
		return err
	}

	err = k.Private.WriteFile(keyFile)
	if err != nil {
		return err
	}

	err = k.Public.WriteFile(publicKeyFile)
	if err != nil {
		return err
	}

	err = ca.WriteFile(certificateFile)
	if err != nil {
		return err
	}

	// output the public ssh key
	ssh, err := publicPEMtoOpenSSH(k.Public.PEMBlock())
	if err != nil {
		return err
	}

	err = os.WriteFile(publicSSHFile, []byte(ssh), os.ModePerm)
	if err != nil {
		return err
	}

	// set the outputs
	p.config.Cert = File{
		Path:      certificateFile,
		Directory: directory,
		Filename:  fmt.Sprintf("%s.cert", p.config.Meta.Name),
		Contents:  ca.String(),
	}

	p.config.PrivateKey = File{
		Path:      keyFile,
		Directory: directory,
		Filename:  fmt.Sprintf("%s.key", p.config.Meta.Name),
		Contents:  k.Private.String(),
	}

	p.config.PublicKeyPEM = File{
		Path:      publicKeyFile,
		Directory: directory,
		Filename:  fmt.Sprintf("%s.pub", p.config.Meta.Name),
		Contents:  k.Public.String(),
	}

	p.config.PublicKeySSH = File{
		Path:      publicSSHFile,
		Directory: directory,
		Filename:  fmt.Sprintf("%s.ssh", p.config.Meta.Name),
		Contents:  ssh,
	}

	return nil
}

func (p *CAProvider) Destroy(ctx context.Context, force bool) error {
	if ctx.Err() != nil {
		p.log.Debug("Context cancelled, skipping destroy", "ref", p.config.Meta.ID)
	}

	p.log.Info("Destroy CA Certificate", "ref", p.config.Meta.ID)

	return destroy(p.config.Meta.Module, p.config.Meta.Name, p.config.Output, p.log)
}

func (p *CAProvider) Lookup() ([]string, error) {
	return nil, nil
}

func (p *CAProvider) Refresh(ctx context.Context) error {
	if ctx.Err() != nil {
		p.log.Debug("Context cancelled, skipping refresh", "ref", p.config.Meta.ID)
		return nil
	}

	p.log.Debug("Refresh CA Certificate", "ref", p.config.Meta.ID)

	return nil
}

func (p *CAProvider) Changed() (bool, error) {
	p.log.Debug("Checking changes Leaf Certificate", "ref", p.config.Meta.ID)

	return false, nil
}

func (p *LeafProvider) Create(ctx context.Context) error {
	if ctx.Err() != nil {
		p.log.Debug("Context cancelled, skipping leaf", "ref", p.config.Meta.ID)
		return nil
	}

	p.log.Info("Creating Leaf Certificate", "ref", p.config.Meta.ID)

	directory := strings.Replace(p.config.Meta.Module, ".", "_", -1)
	directory = path.Join(p.config.Output, directory)
	os.MkdirAll(directory, os.ModePerm)

	keyFile := path.Join(directory, fmt.Sprintf("%s-leaf.key", p.config.Meta.Name))
	pubkeyFile := path.Join(directory, fmt.Sprintf("%s-leaf.pub", p.config.Meta.Name))
	pubsshFile := path.Join(directory, fmt.Sprintf("%s-leaf.ssh", p.config.Meta.Name))
	certFile := path.Join(directory, fmt.Sprintf("%s-leaf.cert", p.config.Meta.Name))

	ca := &crypto.X509{}
	err := ca.ReadFile(p.config.CACert)
	if err != nil {
		return retry.RetryableError(fmt.Errorf("Unable to read root certificate %s: %w", p.config.CACert, err))
	}

	rk := crypto.NewKeyPair()
	err = rk.Private.ReadFile(p.config.CAKey)
	if err != nil {
		return retry.RetryableError(fmt.Errorf("Unable to read root key %s: %w", p.config.CAKey, err))
	}

	k, err := crypto.GenerateKeyPair()
	if err != nil {
		return err
	}

	lc, err := crypto.GenerateLeaf(p.config.Meta.Name, p.config.IPAddresses, p.config.DNSNames, ca, rk.Private, k.Private)
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

	err = os.WriteFile(pubsshFile, []byte(ssh), os.ModePerm)
	if err != nil {
		return err
	}

	// set the outputs
	p.config.PublicKeySSH = File{
		Path:      pubsshFile,
		Directory: directory,
		Filename:  fmt.Sprintf("%s-leaf.ssh", p.config.Meta.Name),
		Contents:  ssh,
	}

	p.config.PublicKeyPEM = File{
		Path:      pubkeyFile,
		Directory: directory,
		Filename:  fmt.Sprintf("%s-leaf.pub", p.config.Meta.Name),
		Contents:  k.Public.String(),
	}

	p.config.Cert = File{
		Path:      certFile,
		Directory: directory,
		Filename:  fmt.Sprintf("%s-leaf.cert", p.config.Meta.Name),
		Contents:  lc.String(),
	}

	p.config.PrivateKey = File{
		Path:      keyFile,
		Directory: directory,
		Filename:  fmt.Sprintf("%s-leaf.key", p.config.Meta.Name),
		Contents:  k.Private.String(),
	}

	return err
}

func (p *LeafProvider) Destroy(ctx context.Context, force bool) error {
	if ctx.Err() != nil {
		p.log.Debug("Context cancelled, skipping destroy", "ref", p.config.Meta.Name)
		return nil
	}

	p.log.Info("Destroy Leaf Certificate", "ref", p.config.Meta.Name)

	return destroy(p.config.Meta.Module, fmt.Sprintf("%s-leaf", p.config.Meta.Name), p.config.Output, p.log)
}

func (p *LeafProvider) Refresh(ctx context.Context) error {
	if ctx.Err() != nil {
		p.log.Debug("Context cancelled, skipping refresh", "ref", p.config.Meta.Name)
		return nil
	}

	p.log.Debug("Refresh Leaf Certificate", "ref", p.config.Meta.Name)

	return nil
}

func (p *LeafProvider) Lookup() ([]string, error) {
	return nil, nil
}

func (p *LeafProvider) Changed() (bool, error) {
	p.log.Debug("Checking changes Leaf Certificate", "ref", p.config.Meta.Name)

	return false, nil
}

func destroy(module, name, output string, log logger.Logger) error {
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

	// if there is a module directory and it is empty, remove it
	if module != "" {
		directory := strings.Replace(module, ".", "_", -1)
		directory = path.Join(output, directory)

		if empty, err := isEmpty(directory); empty && err != nil {
			os.RemoveAll(directory)
		}
	}

	return nil
}

func isEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
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
