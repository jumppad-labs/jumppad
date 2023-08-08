package cert

import (
	"os"
	"path"
	"testing"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/testutils"
	"github.com/stretchr/testify/require"
)

func TestCertCAProcessSetsAbsoluteValues(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	ca := &CertificateCA{
		ResourceMetadata: types.ResourceMetadata{File: "./"},
		Output:           "./output",
	}

	err = ca.Process()
	require.NoError(t, err)

	require.Equal(t, path.Join(wd, "./output"), ca.Output)
}

func TestCertCALoadsValuesFromState(t *testing.T) {
	testutils.SetupState(t, `
{
  "blueprint": null,
  "resources": [
	{
			"id": "resource.certificate_ca.test",
      "name": "test",
      "status": "created",
      "type": "certificate_ca",
			"key_path": "mine.key",
			"cert_path": "mine.cert"
	}
	]
}`)

	ca := &CertificateCA{
		ResourceMetadata: types.ResourceMetadata{
			File: "./",
			ID:   "resource.certificate_ca.test",
		},
		Output: "./output",
	}

	err := ca.Process()
	require.NoError(t, err)

	require.Equal(t, "mine.key", ca.PrivateKey.Filename)
	require.Equal(t, "mine.cert", ca.Cert.Filename)
}

func TestCertLeafProcessSetsAbsoluteValues(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	ca := &CertificateLeaf{
		ResourceMetadata: types.ResourceMetadata{File: "./"},
		CAKey:            "./key.pem",
		CACert:           "./cert.pem",
		Output:           "./output",
	}

	err = ca.Process()
	require.NoError(t, err)

	require.Equal(t, path.Join(wd, "./key.pem"), ca.CAKey)
	require.Equal(t, path.Join(wd, "./cert.pem"), ca.CACert)
	require.Equal(t, path.Join(wd, "./output"), ca.Output)
}

func TestCertLeafLoadsValuesFromState(t *testing.T) {
	testutils.SetupState(t, `
{
  "blueprint": null,
  "resources": [
	{
			"id": "resource.certificate_leaf.test",
      "name": "test",
      "status": "created",
      "type": "certificate_leaf",
			"key_path": "mine.key",
			"cert_path": "mine.cert"
	}
	]
}`)

	ca := &CertificateLeaf{
		ResourceMetadata: types.ResourceMetadata{
			File: "./",
			ID:   "resource.certificate_leaf.test",
		},
		Output: "./output",
	}

	err := ca.Process()
	require.NoError(t, err)

	require.Equal(t, "mine.key", ca.PrivateKey.Filename)
	require.Equal(t, "mine.cert", ca.Cert.Filename)
}
