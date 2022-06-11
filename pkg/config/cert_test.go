package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCreatesCertificateCA(t *testing.T) {
	c := NewCertificateCA("abc")

	assert.Equal(t, "abc", c.Name)
	assert.Equal(t, TypeCertificateCA, c.Type)
}

func TestCertificateCACreatesCorrectly(t *testing.T) {
	c, _ := CreateConfigFromStrings(t, certificateCA)

	cl, err := c.FindResource("certificate_ca.testing")
	assert.NoError(t, err)

	assert.Equal(t, "testing", cl.Info().Name)
	assert.Equal(t, TypeCertificateCA, cl.Info().Type)
	assert.Equal(t, PendingCreation, cl.Info().Status)
}

func TestCertificateCADisabled(t *testing.T) {
	c, _ := CreateConfigFromStrings(t, certificateCADisabled)

	cl, err := c.FindResource("certificate_ca.testing")
	assert.NoError(t, err)

	assert.Equal(t, "testing", cl.Info().Name)
	assert.Equal(t, Disabled, cl.Info().Status)
}

func TestCertificateLeafCreatesCorrectly(t *testing.T) {
	c, _ := CreateConfigFromStrings(t, certificateLeaf)

	cl, err := c.FindResource("certificate_leaf.testing")
	assert.NoError(t, err)

	assert.Equal(t, "testing", cl.Info().Name)
	assert.Equal(t, TypeCertificateLeaf, cl.Info().Type)
	assert.Equal(t, PendingCreation, cl.Info().Status)

	assert.Equal(t, cl.(*CertificateLeaf).DNSNames[0], "1")
	assert.Equal(t, cl.(*CertificateLeaf).DNSNames[1], "2")

	assert.Equal(t, cl.(*CertificateLeaf).IPAddresses[0], "a")
	assert.Equal(t, cl.(*CertificateLeaf).IPAddresses[1], "b")
}

func TestCertificateLeafDisabled(t *testing.T) {
	c, _ := CreateConfigFromStrings(t, certificateLeafDisabled)

	cl, err := c.FindResource("certificate_leaf.testing")
	assert.NoError(t, err)

	assert.Equal(t, "testing", cl.Info().Name)
	assert.Equal(t, Disabled, cl.Info().Status)
}

const certificateCA = `
certificate_ca "testing" {
	output = "/"
}
`

const certificateCADisabled = `
certificate_ca "testing" {
	disabled = true
	output = "/"
}
`
const certificateLeaf = `
certificate_leaf "testing" {
	ip_addresses = ["a","b"]
	dns_names = ["1","2"]
	output = "/"
	ca_cert = "./file"
	ca_key = "./file"
}
`

const certificateLeafDisabled = `
certificate_leaf "testing" {
	ca_cert = "./file"
	ca_key = "./file"
	disabled = true
	ip_addresses = ["a","b"]
	dns_names = ["1","2"]
	output = "/"
}
`
