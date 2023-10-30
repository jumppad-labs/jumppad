package exec

import (
	"testing"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/testutils"
	"github.com/stretchr/testify/require"
)

func TestExecSetsOutputsFromState(t *testing.T) {
	testutils.SetupState(t, `
{
  "blueprint": null,
  "resources": [
	{
			"id": "resource.exec.test",
      "name": "test",
      "status": "created",
      "type": "exec",
			"pid": 42
	}
	]
}`)

	c := &Exec{
		ResourceMetadata: types.ResourceMetadata{
			ID: "resource.exec.test",
		},
	}

	c.Process()

	require.Equal(t, 42, c.PID)
}
