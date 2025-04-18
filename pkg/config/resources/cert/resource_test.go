package cert

import (
	"os"
	"path"
	"testing"

	"github.com/instruqt/jumppad/pkg/config"
	"github.com/instruqt/jumppad/testutils"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/stretchr/testify/require"
)

func init() {
	config.RegisterResource(TypeCertificateCA, &CertificateCA{}, &CAProvider{})
	config.RegisterResource(TypeCertificateLeaf, &CertificateLeaf{}, &LeafProvider{})
}

func TestCertCAProcessSetsAbsoluteValues(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	ca := &CertificateCA{
		ResourceBase: types.ResourceBase{Meta: types.Meta{File: "./"}},
		Output:       "./output",
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
			"meta": {
				"id": "resource.certificate_ca.test",
				"name": "test",
				"type": "certificate_ca"
			},
			"private_key": {
				"filename": "private.key"
			},
			"public_key_pem": {
				"filename": "public.key"
			},
			"public_key_ssh": {
				"filename": "public.ssh"
			},
			"certificate": {
				"filename": "cert.pem"
			}
		}
	]
}`)

	ca := &CertificateCA{
		ResourceBase: types.ResourceBase{
			Meta: types.Meta{
				File: "./",
				ID:   "resource.certificate_ca.test",
			},
		},
		Output: "./output",
	}

	err := ca.Process()
	require.NoError(t, err)

	require.Equal(t, "private.key", ca.PrivateKey.Filename)
	require.Equal(t, "public.key", ca.PublicKeyPEM.Filename)
	require.Equal(t, "public.ssh", ca.PublicKeySSH.Filename)
	require.Equal(t, "cert.pem", ca.Cert.Filename)
}

func TestCertLeafProcessSetsAbsoluteValues(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	ca := &CertificateLeaf{
		ResourceBase: types.ResourceBase{Meta: types.Meta{File: "./"}},
		CAKey:        "./key.pem",
		CACert:       "./cert.pem",
		Output:       "./output",
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
			"meta": {
				"id": "resource.certificate_leaf.test",
  	    "name": "test",
  	    "type": "certificate_leaf"
			},
			"private_key": {
				"filename": "private.key"
			},
			"public_key_pem": {
				"filename": "public.key"
			},
			"public_key_ssh": {
				"filename": "public.ssh"
			},
			"certificate": {
				"filename": "cert.pem"
			}
	}
	]
}`)

	ca := &CertificateLeaf{
		ResourceBase: types.ResourceBase{
			Meta: types.Meta{
				File: "./",
				ID:   "resource.certificate_leaf.test",
			},
		},
		Output: "./output",
	}

	err := ca.Process()
	require.NoError(t, err)

	require.Equal(t, "private.key", ca.PrivateKey.Filename)
	require.Equal(t, "cert.pem", ca.Cert.Filename)
}
