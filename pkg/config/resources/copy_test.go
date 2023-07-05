package resources

import (
	"os"
	"testing"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/stretchr/testify/require"
)

func TestCopyProcessSetsAbsolute(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	c := &Copy{
		ResourceMetadata: types.ResourceMetadata{File: "./"},
		Source:           "./",
		Destination:      "./",
	}

	c.Process()

	require.Equal(t, wd, c.Source)
	require.Equal(t, wd, c.Destination)
}

func TestCopySetsOutputsFromState(t *testing.T) {
	setupState(t, `
{
  "blueprint": null,
  "resources": [
	{
			"id": "resource.copy.test",
      "name": "test",
      "status": "created",
      "type": "copy",
			"copied_files": ["a","b"]
	}
	]
}`)

	c := &Copy{
		ResourceMetadata: types.ResourceMetadata{
			ID:   "resource.copy.test",
			File: "./",
		},
		Source:      "./",
		Destination: "./",
	}

	c.Process()

	require.Equal(t, []string{"a", "b"}, c.CopiedFiles)

}
