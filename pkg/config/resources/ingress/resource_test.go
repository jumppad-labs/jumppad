package ingress

import (
	"testing"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/testutils"
	"github.com/stretchr/testify/require"
)

func init() {
	config.RegisterResource(TypeIngress, &Ingress{}, &Provider{})
}

func TestIngressSetsOutputsFromState(t *testing.T) {
	testutils.SetupState(t, `
{
  "blueprint": null,
  "resources": [
	{
			"resource_id": "resource.ingress.test",
      "resource_name": "test",
      "resource_type": "ingress",
			"ingress_id": "42",
			"local_address": "127.0.0.1"
	}
	]
}`)

	c := &Ingress{
		ResourceMetadata: types.ResourceMetadata{
			ResourceID: "resource.ingress.test",
		},
	}

	c.Process()

	require.Equal(t, "42", c.IngressID)
	require.Equal(t, "127.0.0.1", c.LocalAddress)
}
