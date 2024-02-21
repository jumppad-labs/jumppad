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
      "meta": {
      	"id": "resource.local_exec.test",
      	"name": "test",
      	"type": "local_exec"
			},
			"pid": 42
	}
	]
}`)

	c := &LocalExec{
		ResourceBase: types.ResourceBase{
			Meta: types.Meta{
				ID: "resource.local_exec.test",
			},
		},
	}

	c.Process()

	require.Equal(t, 42, c.Pid)
}
