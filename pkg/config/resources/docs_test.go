package resources

import (
	"os"
	"testing"

	"github.com/shipyard-run/hclconfig/types"
	"github.com/stretchr/testify/require"
)

func TestDocsProcessSetsAbsolute(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	h := &Docs{
		ResourceMetadata: types.ResourceMetadata{File: "./"},
		Path:             "./",
	}

	err = h.Process()
	require.NoError(t, err)

	require.Equal(t, wd, h.Path)
}

func TestDocsLoadsValuesFromState(t *testing.T) {
	setupState(t, `
{
  "blueprint": null,
  "resources": [
	{
			"id": "resource.docs.test",
      "name": "test",
      "status": "created",
      "type": "docs",
			"fqdn": "fqdn.mine"
	}
	]
}`)

	docs := &Docs{
		ResourceMetadata: types.ResourceMetadata{
			File: "./",
			ID:   "resource.docs.test",
		},
	}

	err := docs.Process()
	require.NoError(t, err)

	require.Equal(t, "fqdn.mine", docs.FQDN)
}
