package resources

import (
	"testing"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/testutils"
	"github.com/stretchr/testify/require"
)

func TestLocalExecSetsOutputsFromState(t *testing.T) {
	testutils.SetupState(t, `
{
  "blueprint": null,
  "resources": [
	{
			"id": "resource.local_exec.test",
      "name": "test",
      "status": "created",
      "type": "local_exec",
			"pid": 42
	}
	]
}`)

	c := &LocalExec{
		ResourceMetadata: types.ResourceMetadata{
			ID: "resource.local_exec.test",
		},
	}

	c.Process()

	require.Equal(t, 42, c.Pid)
}
