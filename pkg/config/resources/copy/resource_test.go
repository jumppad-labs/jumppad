package copy

import (
	"os"
	"testing"

	"github.com/instruqt/jumppad/pkg/config"
	"github.com/instruqt/jumppad/testutils"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/stretchr/testify/require"
)

func init() {
	config.RegisterResource(TypeCopy, &Copy{}, &Provider{})
}

func TestCopyProcessSetsAbsoluteIfLocal(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	c := &Copy{
		ResourceBase: types.ResourceBase{Meta: types.Meta{File: "./"}},
		Source:       "./",
		Destination:  "./",
	}

	c.Process()

	require.Equal(t, wd, c.Source)
	require.Equal(t, wd, c.Destination)
}

func TestCopyProcessSetsAbsoluteIfNotLocal(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	c := &Copy{
		ResourceBase: types.ResourceBase{Meta: types.Meta{File: "./"}},
		Source:       "github.com/instruqt/jumppad",
		Destination:  "./",
	}

	c.Process()

	require.Equal(t, "github.com/instruqt/jumppad", c.Source)
	require.Equal(t, wd, c.Destination)
}

func TestCopySetsOutputsFromState(t *testing.T) {
	testutils.SetupState(t, `
{
  "blueprint": null,
  "resources": [
	{
			"meta": {
				"id": "resource.copy.test",
  	    "name": "test",
  	    "type": "copy"
			},
			"copied_files": ["a","b"]
	}
	]
}`)

	c := &Copy{
		ResourceBase: types.ResourceBase{
			Meta: types.Meta{
				ID:   "resource.copy.test",
				File: "./",
			},
		},
		Source:      "./",
		Destination: "./",
	}

	c.Process()

	require.Equal(t, []string{"a", "b"}, c.CopiedFiles)
}
