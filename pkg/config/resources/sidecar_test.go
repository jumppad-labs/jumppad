package resources

import (
	"os"
	"testing"

	"github.com/shipyard-run/hclconfig/types"
	"github.com/stretchr/testify/require"
)

func TestSidecarProcessSetsAbsolute(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	c := &Sidecar{
		ResourceMetadata: types.ResourceMetadata{File: "./"},
		Volumes: []Volume{
			{
				Source:      "./",
				Destination: "./",
			},
		},
	}

	c.Process()

	require.Equal(t, wd, c.Volumes[0].Source)
}
