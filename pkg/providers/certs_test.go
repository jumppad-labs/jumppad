package providers

import (
	"fmt"
	"path"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/config/resources"
	"github.com/stretchr/testify/require"
)

func setupCACert(t *testing.T) (*resources.CertificateCA, *CertificateCA) {
	dir := t.TempDir()

	cc := config.NewCertificateCA("test")
	cc.Output = dir

	p := NewCertificateCA(cc, hclog.NewNullLogger())

	return cc, p
}

func setupLeafCert(t *testing.T) (*config.CertificateLeaf, *CertificateLeaf) {
	dir := t.TempDir()

	cc := config.NewCertificateCA("root")
	p := NewCertificateCA(cc, hclog.NewNullLogger())

	cc.Output = dir
	err := p.Create()
	require.NoError(t, err)

	cl := config.NewCertificateLeaf("test")
	cl.Output = dir
	cl.IPAddresses = []string{"127.0.0.1"}
	cl.DNSNames = []string{"localhost"}
	cl.CACert = path.Join(dir, "root.cert")
	cl.CAKey = path.Join(dir, "root.key")

	pl := NewCertificateLeaf(cl, hclog.NewNullLogger())

	return cl, pl
}

func TestGeneratesValidCA(t *testing.T) {
	c, p := setupCACert(t)

	err := p.Create()
	require.NoError(t, err)

	require.FileExists(t, path.Join(c.Output, fmt.Sprintf("%s.cert", c.Name)))
	require.FileExists(t, path.Join(c.Output, fmt.Sprintf("%s.key", c.Name)))
}

func TestDestroyCleansUpCA(t *testing.T) {
	c, p := setupCACert(t)

	err := p.Create()
	require.NoError(t, err)

	err = p.Destroy()
	require.NoError(t, err)

	require.NoFileExists(t, path.Join(c.Output, fmt.Sprintf("%s.cert", c.Name)))
	require.NoFileExists(t, path.Join(c.Output, fmt.Sprintf("%s.key", c.Name)))
}

func TestGeneratesValidLeaf(t *testing.T) {
	c, p := setupLeafCert(t)

	err := p.Create()
	require.NoError(t, err)

	require.FileExists(t, path.Join(c.Output, fmt.Sprintf("%s.cert", c.Name)))
	require.FileExists(t, path.Join(c.Output, fmt.Sprintf("%s.key", c.Name)))
}

func TestDestroyCleansUpLeaf(t *testing.T) {
	c, p := setupLeafCert(t)

	err := p.Create()
	require.NoError(t, err)

	err = p.Destroy()
	require.NoError(t, err)

	require.NoFileExists(t, path.Join(c.Output, fmt.Sprintf("%s.cert", c.Name)))
	require.NoFileExists(t, path.Join(c.Output, fmt.Sprintf("%s.key", c.Name)))
}
