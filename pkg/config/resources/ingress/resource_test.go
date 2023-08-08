package ingress

import (
	"testing"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/testutils"
	"github.com/stretchr/testify/require"
)

func TestIngressSetsOutputsFromState(t *testing.T) {
	testutils.SetupState(t, `
{
  "blueprint": null,
  "resources": [
	{
			"id": "resource.ingress.test",
      "name": "test",
      "status": "created",
      "type": "ingress",
			"ingress_id": "42",
			"address": "127.0.0.1"
	}
	]
}`)

	c := &Ingress{
		ResourceMetadata: types.ResourceMetadata{
			ID: "resource.ingress.test",
		},
	}

	c.Process()

	require.Equal(t, "42", c.IngressID)
	require.Equal(t, "127.0.0.1", c.Address)
}
