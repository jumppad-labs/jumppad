package cert

import (
	"context"
	"fmt"
	"path"
	"testing"

	"github.com/instruqt/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/stretchr/testify/require"
)

func setupCACert(t *testing.T) (*CertificateCA, *CAProvider) {
	dir := t.TempDir()

	ca := &CertificateCA{ResourceBase: types.ResourceBase{Meta: types.Meta{Name: "test"}}}
	ca.Output = dir

	p := &CAProvider{ca, logger.NewTestLogger(t)}

	return ca, p
}

func setupLeafCert(t *testing.T) (*CertificateLeaf, *LeafProvider) {
	dir := t.TempDir()
	ca, p := setupCACert(t)

	err := p.Create(context.Background())
	require.NoError(t, err)

	cl := &CertificateLeaf{ResourceBase: types.ResourceBase{Meta: types.Meta{Name: "test"}}}
	cl.Output = dir
	cl.IPAddresses = []string{"127.0.0.1"}
	cl.DNSNames = []string{"localhost"}
	cl.CACert = ca.Cert.Path
	cl.CAKey = ca.PrivateKey.Path

	pl := &LeafProvider{cl, logger.NewTestLogger(t)}

	return cl, pl
}

func TestGeneratesValidCA(t *testing.T) {
	c, p := setupCACert(t)

	err := p.Create(context.Background())
	require.NoError(t, err)

	require.FileExists(t, path.Join(c.Output, fmt.Sprintf("%s.cert", c.Meta.Name)))
	require.FileExists(t, path.Join(c.Output, fmt.Sprintf("%s.key", c.Meta.Name)))
	require.FileExists(t, path.Join(c.Output, fmt.Sprintf("%s.pub", c.Meta.Name)))
	require.FileExists(t, path.Join(c.Output, fmt.Sprintf("%s.ssh", c.Meta.Name)))
}

func TestDestroyCleansUpCA(t *testing.T) {
	c, p := setupCACert(t)

	err := p.Create(context.Background())
	require.NoError(t, err)

	err = p.Destroy(context.Background(), false)
	require.NoError(t, err)

	require.NoFileExists(t, path.Join(c.Output, fmt.Sprintf("%s.cert", c.Meta.Name)))
	require.NoFileExists(t, path.Join(c.Output, fmt.Sprintf("%s.key", c.Meta.Name)))
	require.NoFileExists(t, path.Join(c.Output, fmt.Sprintf("%s.pub", c.Meta.Name)))
	require.NoFileExists(t, path.Join(c.Output, fmt.Sprintf("%s.ssh", c.Meta.Name)))
}

func TestGeneratesValidLeaf(t *testing.T) {
	c, p := setupLeafCert(t)

	err := p.Create(context.Background())
	require.NoError(t, err)

	require.FileExists(t, path.Join(c.Output, fmt.Sprintf("%s-leaf.cert", c.Meta.Name)))
	require.FileExists(t, path.Join(c.Output, fmt.Sprintf("%s-leaf.key", c.Meta.Name)))
	require.FileExists(t, path.Join(c.Output, fmt.Sprintf("%s-leaf.pub", c.Meta.Name)))
	require.FileExists(t, path.Join(c.Output, fmt.Sprintf("%s-leaf.ssh", c.Meta.Name)))
}

func TestDestroyCleansUpLeaf(t *testing.T) {
	c, p := setupLeafCert(t)

	err := p.Create(context.Background())
	require.NoError(t, err)

	err = p.Destroy(context.Background(), false)
	require.NoError(t, err)

	require.NoFileExists(t, path.Join(c.Output, fmt.Sprintf("%s-leaf.cert", c.Meta.Name)))
	require.NoFileExists(t, path.Join(c.Output, fmt.Sprintf("%s-leaf.key", c.Meta.Name)))
	require.NoFileExists(t, path.Join(c.Output, fmt.Sprintf("%s-leaf.pub", c.Meta.Name)))
	require.NoFileExists(t, path.Join(c.Output, fmt.Sprintf("%s-leaf.ssh", c.Meta.Name)))
}
