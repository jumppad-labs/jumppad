package resources

import "github.com/shipyard-run/hclconfig/types"

// TypeCertificateCA is the resource string for a self-signed CA
const TypeCertificateCA string = "certificate_ca"

// CertificateCA allows the generate of CA certificates
type CertificateCA struct {
	types.ResourceMetadata `hcl:",remain"`

	Output string `hcl:"output" json:"output"`
}

func (c *CertificateCA) Process() error {
	c.Output = ensureAbsolute(c.Output, c.File)

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
}

func (c *CertificateLeaf) Process() error {
	c.CACert = ensureAbsolute(c.CACert, c.File)
	c.CAKey = ensureAbsolute(c.CAKey, c.File)
	c.Output = ensureAbsolute(c.Output, c.File)

	return nil
}
