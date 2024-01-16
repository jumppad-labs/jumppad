package container

import (
	"os"
	"testing"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/stretchr/testify/require"
)

func TestContainerProcessSetsAbsolute(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	c := &Container{
		ResourceMetadata: types.ResourceMetadata{ResourceFile: "./"},
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
