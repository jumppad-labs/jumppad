package docs

import (
	"testing"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/testutils"
	"github.com/stretchr/testify/require"
)

func init() {
	config.RegisterResource(TypeDocs, &Docs{}, &DocsProvider{})
}

func TestDocsProcessSetsAbsolute(t *testing.T) {
	h := &Docs{
		ResourceMetadata: types.ResourceMetadata{ResourceFile: "./"},
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
			"resource_id": "resource.docs.test",
      "resource_name": "test",
      "resource_type": "docs",
			"fqdn": "fqdn.mine"
	}
	]
}`)

	docs := &Docs{
		ResourceMetadata: types.ResourceMetadata{
			ResourceFile: "./",
			ResourceID:   "resource.docs.test",
		},
	}

	err := docs.Process()
	require.NoError(t, err)

	require.Equal(t, "fqdn.mine", docs.ContainerName)
}
