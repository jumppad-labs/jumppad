package config

// TypeCertificateCA is the resource string for a self-signed CA
const TypeCertificateCA ResourceType = "certificate_ca"

// CertificateCA allows the generate of CA certificates
type CertificateCA struct {
	ResourceInfo `hcl:",remain" mapstructure:",squash"`

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	Output string `hcl:"output" json:"output"`
}

// NewCertificateCA creates a new CA certificate config resource
func NewCertificateCA(name string) *CertificateCA {
	return &CertificateCA{ResourceInfo: ResourceInfo{Name: name, Type: TypeCertificateCA, Status: PendingCreation}}
}

// TypeCertificateCA is the resource string for a self-signed CA
const TypeCertificateLeaf ResourceType = "certificate_leaf"

// CertificateCA allows the generate of CA certificates
type CertificateLeaf struct {
	ResourceInfo `hcl:",remain" mapstructure:",squash"`

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	CAKey  string `hcl:"ca_key" json:"ca_key"`   // Path to the primary key for the root CA
	CACert string `hcl:"ca_cert" json:"ca_cert"` // Path to the root CA

	IPAddresses []string `hcl:"ip_addresses,optional" json:"ip_addresses,omitempty" mapstructure:"ip_addresses"` // ip addresses to add to the cert
	DNSNames    []string `hcl:"dns_names,optional" json:"dns_names,omitempty" mapstructure:"dns_names"`          // DNS names to add to the cert

	Output string `hcl:"output" json:"output"` // output location for the certificate
}

// NewCertificateLeaf creates a new Leaf certificate resource
func NewCertificateLeaf(name string) *CertificateLeaf {
	return &CertificateLeaf{ResourceInfo: ResourceInfo{Name: name, Type: TypeCertificateLeaf, Status: PendingCreation}}
}
