package exec

import (
	"testing"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/testutils"
	"github.com/stretchr/testify/require"
)

func init() {
	config.RegisterResource(TypeLocalExec, &LocalExec{}, &LocalProvider{})
}

func TestLocalExecSetsOutputsFromState(t *testing.T) {
	testutils.SetupState(t, `
{
  "blueprint": null,
  "resources": [
	{
			"resource_id": "resource.local_exec.test",
      "resource_name": "test",
      "resource_type": "local_exec",
			"pid": 42
	}
	]
}`)

	c := &LocalExec{
		ResourceMetadata: types.ResourceMetadata{
			ResourceID: "resource.local_exec.test",
		},
	}

	c.Process()

	require.Equal(t, 42, c.Pid)
}
