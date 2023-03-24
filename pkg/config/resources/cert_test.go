package resources

import (
	"os"
	"path"
	"testing"

	"github.com/shipyard-run/hclconfig/types"
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
