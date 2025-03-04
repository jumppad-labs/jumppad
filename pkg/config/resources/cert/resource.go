package cert

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// TypeCertificateCA is the resource string for a self-signed CA
const TypeCertificateCA string = "certificate_ca"

/*
CertificateCA allows the generate of CA certificates

@resource
*/
type CertificateCA struct {
	/*
	 embedded type holding name, etc

	 @ignore
	*/
	types.ResourceBase `hcl:",remain"`

	// Output directory to write the certificate and key too
	Output string `hcl:"output" json:"output"`

	/*
		Key is the value related to the certificate key

		@computed
	*/
	PrivateKey File `hcl:"private_key,optional" json:"private_key"`
	/*
		Key is the value related to the certificate key

		@computed
	*/
	PublicKeyPEM File `hcl:"public_key_pem,optional" json:"public_key_pem"`
	/*
		Key is the value related to the certificate key

		@computed
	*/
	PublicKeySSH File `hcl:"public_key_ssh,optional" json:"public_key_ssh"`

	/*
		Cert is the value related to the certificate

		@computed
	*/
	Cert File `hcl:"certificate,optional" json:"certificate"`
}

func (c *CertificateCA) Process() error {
	c.Output = utils.EnsureAbsolute(c.Output, c.Meta.File)
	c.PrivateKey = File{}
	c.PublicKeySSH = File{}
	c.PublicKeyPEM = File{}
	c.Cert = File{}

	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(c.Meta.ID)
		if r != nil {
			kstate := r.(*CertificateCA)
			c.PrivateKey = kstate.PrivateKey
			c.PublicKeySSH = kstate.PublicKeySSH
			c.PublicKeyPEM = kstate.PublicKeyPEM
			c.Cert = kstate.Cert
		}
	}

	return nil
}

// TypeCertificateCA is the resource string for a self-signed CA
const TypeCertificateLeaf string = "certificate_leaf"

// CertificateCA allows the generate of CA certificates
type CertificateLeaf struct {
	types.ResourceBase `hcl:",remain"`

	CAKey  string `hcl:"ca_key" json:"ca_key"`   // Path to the primary key for the root CA
	CACert string `hcl:"ca_cert" json:"ca_cert"` // Path to the root CA

	IPAddresses []string `hcl:"ip_addresses,optional" json:"ip_addresses,omitempty"` // ip addresses to add to the cert
	DNSNames    []string `hcl:"dns_names,optional" json:"dns_names,omitempty"`       // DNS names to add to the cert

	Output string `hcl:"output" json:"output"` // output location for the certificate

	// output parameters

	// Key is the value related to the certificate key
	PrivateKey File `hcl:"private_key,optional" json:"private_key"`

	// Key is the value related to the certificate key
	PublicKeyPEM File `hcl:"public_key_pem,optional" json:"public_key_pem"`
	PublicKeySSH File `hcl:"public_key_ssh,optional" json:"public_key_ssh"`

	// Cert is the value related to the certificate
	Cert File `hcl:"certificate,optional" json:"certificate"`
}

func (c *CertificateLeaf) Process() error {
	c.CACert = utils.EnsureAbsolute(c.CACert, c.Meta.File)
	c.CAKey = utils.EnsureAbsolute(c.CAKey, c.Meta.File)
	c.Output = utils.EnsureAbsolute(c.Output, c.Meta.File)
	c.PrivateKey = File{}
	c.PublicKeySSH = File{}
	c.PublicKeyPEM = File{}
	c.Cert = File{}

	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(c.Meta.ID)
		if r != nil {
			kstate := r.(*CertificateLeaf)
			c.PrivateKey = kstate.PrivateKey
			c.PublicKeySSH = kstate.PublicKeySSH
			c.PublicKeyPEM = kstate.PublicKeyPEM
			c.Cert = kstate.Cert
		}
	}

	return nil
}

type File struct {
	Filename  string `hcl:"filename,optional" json:"filename"`
	Directory string `hcl:"directory,optional" json:"directory"`
	Path      string `hcl:"path,optional" json:"path"`
	Contents  string `hcl:"contents,optional" json:"contents"`
}
