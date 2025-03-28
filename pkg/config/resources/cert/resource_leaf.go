package cert

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// TypeCertificateCA is the resource string for a self-signed CA
const TypeCertificateLeaf string = "certificate_leaf"

/*
CertificateLeaf generates leaf certificates

```hcl

	resource "certificate_leaf" "name" {
	  ...
	}

```

@include cert.File

@resource
*/
type CertificateLeaf struct {
	/*
	 embedded type holding name, etc

	 @ignore
	*/
	types.ResourceBase `hcl:",remain"`

	/*
		Path to the primary key for the root CA

		```hcl
		ca_key = resource.certificate_ca.root.private_key.path
		```
	*/
	CAKey string `hcl:"ca_key" json:"ca_key"`
	/*
		Path to the root CA

		```hcl
		ca_cert = resource.certificate_ca.root.certificate.path
		```
	*/
	CACert string `hcl:"ca_cert" json:"ca_cert"`
	/*
		IP addresses to add to the cert.

		```hcl
		ip_addresses = ["127.0.0.1"]
		```
	*/
	IPAddresses []string `hcl:"ip_addresses,optional" json:"ip_addresses,omitempty"`
	/*
		DNS names to add to the cert.

		```hcl
		dns_names = [
		  "localhost",
		  "localhost:30090",
		  "30090",
		  "connector",
		  "connector",
		]
		```
	*/
	DNSNames []string `hcl:"dns_names,optional" json:"dns_names,omitempty"`
	/*
		Output directory to write the certificate and key to.

		```hcl
		output = data("certs")
		```
	*/
	Output string `hcl:"output" json:"output"`
	/*
		Key is the value related to the certificate key

		@computed
	*/
	PrivateKey File `hcl:"private_key,optional" json:"private_key"`
	/*
		The PEM key value of the generated certificate.

		@computed
	*/
	PublicKeyPEM File `hcl:"public_key_pem,optional" json:"public_key_pem"`
	/*
		The SSH key value of the generated certificate.

		@computed
	*/
	PublicKeySSH File `hcl:"public_key_ssh,optional" json:"public_key_ssh"`
	/*
		The generated certificate.
	*/
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
