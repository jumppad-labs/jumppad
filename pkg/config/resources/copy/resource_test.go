package copy

import (
	"os"
	"testing"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/testutils"
	"github.com/stretchr/testify/require"
)

func init() {
	config.RegisterResource(TypeCopy, &Copy{}, &Provider{})
}

func TestCopyProcessSetsAbsolute(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	c := &Copy{
		ResourceMetadata: types.ResourceMetadata{ResourceFile: "./"},
		Source:           "./",
		Destination:      "./",
	}

	c.Process()

	require.Equal(t, wd, c.Source)
	require.Equal(t, wd, c.Destination)
}

func TestCopySetsOutputsFromState(t *testing.T) {
	testutils.SetupState(t, `
{
  "blueprint": null,
  "resources": [
	{
			"resource_id": "resource.copy.test",
      "resource_name": "test",
      "resource_type": "copy",
			"copied_files": ["a","b"]
	}
	]
}`)

	c := &Copy{
		ResourceMetadata: types.ResourceMetadata{
			ResourceID:   "resource.copy.test",
			ResourceFile: "./",
		},
		Source:      "./",
		Destination: "./",
	}

	c.Process()

	require.Equal(t, []string{"a", "b"}, c.CopiedFiles)

}
