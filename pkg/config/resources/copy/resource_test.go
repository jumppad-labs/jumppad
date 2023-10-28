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
		ResourceMetadata: types.ResourceMetadata{File: "./"},
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
