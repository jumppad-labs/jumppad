package resources

import (
	"os"
	"testing"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/stretchr/testify/require"
)

func TestSidecarProcessSetsAbsolute(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	c := &Sidecar{
		ResourceMetadata: types.ResourceMetadata{File: "./"},
		Volumes: []Volume{
			{
				Source:      "./",
				Destination: "./",
			},
		},
	}

	c.Process()

	require.Equal(t, wd, c.Volumes[0].Source)
}

func TestSidecarLoadsValuesFromState(t *testing.T) {
	setupState(t, `
{
  "blueprint": null,
  "resources": [
	{
			"id": "resource.sidecar.test",
      "name": "test",
      "status": "created",
      "type": "sidecar",
			"fqdn": "fqdn.mine"
	}
	]
}`)

	docs := &Sidecar{
		ResourceMetadata: types.ResourceMetadata{
			File: "./",
			ID:   "resource.sidecar.test",
		},
	}

	err := docs.Process()
	require.NoError(t, err)

	require.Equal(t, "fqdn.mine", docs.FQDN)
}
