package resources

import "github.com/shipyard-run/hclconfig/types"

// TypeCertificateCA is the resource string for a self-signed CA
const TypeCertificateCA string = "certificate_ca"

// CertificateCA allows the generate of CA certificates
type CertificateCA struct {
	types.ResourceMetadata `hcl:",remain"`

	// Output directory to write the certificate and key too
	Output string `hcl:"output" json:"output"`

	// output parameters

	// Key is the value related to the certificate key
	PrivateKey *File `hcl:"private_key,block" json:"private_key"`

	// Key is the value related to the certificate key
	PublicKey *File `hcl:"public_key,block" json:"public_key"`

	// Cert is the value related to the certificate
	Cert *File `hcl:"certificate,block" json:"cert"`
}

func (c *CertificateCA) Process() error {
	c.Output = ensureAbsolute(c.Output, c.File)
	c.PrivateKey = &File{}
	c.PublicKey = &File{}
	c.Cert = &File{}

	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(c.ID)
		if r != nil {
			kstate := r.(*CertificateCA)
			c.PrivateKey = kstate.PrivateKey
			c.PublicKey = kstate.PublicKey
			c.Cert = kstate.Cert
		}
	}

	return nil
}

// TypeCertificateCA is the resource string for a self-signed CA
const TypeCertificateLeaf string = "certificate_leaf"

// CertificateCA allows the generate of CA certificates
type CertificateLeaf struct {
	types.ResourceMetadata `hcl:",remain"`

	CAKey  string `hcl:"ca_key" json:"ca_key"`   // Path to the primary key for the root CA
	CACert string `hcl:"ca_cert" json:"ca_cert"` // Path to the root CA

	IPAddresses []string `hcl:"ip_addresses,optional" json:"ip_addresses,omitempty"` // ip addresses to add to the cert
	DNSNames    []string `hcl:"dns_names,optional" json:"dns_names,omitempty"`       // DNS names to add to the cert

	Output string `hcl:"output" json:"output"` // output location for the certificate

	// output parameters

	// Key is the value related to the certificate key
	PrivateKey *File `hcl:"private_key,block" json:"private_key"`

	// Key is the value related to the certificate key
	PublicKey *File `hcl:"public_key,block" json:"public_key"`

	// Cert is the value related to the certificate
	Cert *File `hcl:"certificate,block" json:"cert"`
}

func (c *CertificateLeaf) Process() error {
	c.CACert = ensureAbsolute(c.CACert, c.File)
	c.CAKey = ensureAbsolute(c.CAKey, c.File)
	c.Output = ensureAbsolute(c.Output, c.File)

	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(c.ID)
		if r != nil {
			kstate := r.(*CertificateLeaf)
			c.PrivateKey = kstate.PrivateKey
			c.PublicKey = kstate.PublicKey
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
