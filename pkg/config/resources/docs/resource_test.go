package docs

import (
	"testing"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/testutils"
	"github.com/stretchr/testify/require"
)

func TestDocsProcessSetsAbsolute(t *testing.T) {
	h := &Docs{
		ResourceMetadata: types.ResourceMetadata{File: "./"},
	}

	err := h.Process()
	require.NoError(t, err)
}

func TestDocsLoadsValuesFromState(t *testing.T) {
	testutils.SetupState(t, `
{
  "blueprint": null,
  "resources": [
	{
			"id": "resource.docs.test",
      "name": "test",
      "status": "created",
      "type": "docs",
			"fqrn": "fqdn.mine"
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

	require.Equal(t, "fqdn.mine", docs.ContainerName)
}
