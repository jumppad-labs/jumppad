package cert

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// TypeCertificateCA is the resource string for a self-signed CA
const TypeCertificateCA string = "certificate_ca"

/*
CertificateCA generates CA certificates.

```hcl

	resource "certificate_ca" "name" {
	  ...
	}

```

@include cert.File

@resource
*/
type CertificateCA struct {
	/*
	 embedded type holding name, etc

	 @ignore
	*/
	types.ResourceBase `hcl:",remain"`

	/*
		Output directory to write the certificate and key to.

		```hcl
		output = data("certs")
		```
	*/
	Output string `hcl:"output" json:"output"`

	/*
		The private key of the generated certificate.

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

		@computed
	*/
	Cert File `hcl:"certificate,optional" json:"certificate"`
}

/*
```hcl

	file {
	  ...
	}

```
*/
type File struct {
	/*
		The name of the file.
	*/
	Filename string `hcl:"filename,optional" json:"filename"`
	/*
		The directory the file is written to.
	*/
	Directory string `hcl:"directory,optional" json:"directory"`
	/*
		The full path where the file is written to.
	*/
	Path string `hcl:"path,optional" json:"path"`
	/*
		The contents of the file.
	*/
	Contents string `hcl:"contents,optional" json:"contents"`
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
