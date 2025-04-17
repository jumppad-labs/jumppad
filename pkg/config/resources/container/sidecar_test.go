package container

import (
	"os"
	"testing"

	"github.com/instruqt/jumppad/pkg/config"
	"github.com/instruqt/jumppad/testutils"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/stretchr/testify/require"
)

func init() {
	config.RegisterResource(TypeSidecar, &Sidecar{}, &Provider{})
}

func TestSidecarProcessSetsAbsolute(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	c := &Sidecar{
		ResourceBase: types.ResourceBase{Meta: types.Meta{File: "./"}},
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
	testutils.SetupState(t, `
{
  "blueprint": null,
  "resources": [
	{
			"meta": {
				"id": "resource.sidecar.test",
  	    "name": "test",
  	    "type": "sidecar"
			},
			"container_name": "fqdn.mine"
	}
	]
}`)

	docs := &Sidecar{
		ResourceBase: types.ResourceBase{
			Meta: types.Meta{
				File: "./",
				ID:   "resource.sidecar.test",
			},
		},
	}

	err := docs.Process()
	require.NoError(t, err)

	require.Equal(t, "fqdn.mine", docs.ContainerName)
}
