package resources

import (
	"os"
	"path"
	"testing"

	"github.com/shipyard-run/hclconfig/types"
	"github.com/stretchr/testify/require"
)

func TestContainerProcessSetsAbsolute(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	c := &Container{
		ResourceMetadata: types.ResourceMetadata{File: "./"},
		Volumes: []Volume{
			{
				Source:      "./",
				Destination: "./",
			},
		},
		Build: &Build{
			File:    "./Dockerfile",
			Context: "./",
		},
	}

	c.Process()

	require.Equal(t, wd, c.Volumes[0].Source)

	require.Equal(t, path.Join(wd, "Dockerfile"), c.Build.File)
	require.Equal(t, wd, c.Build.Context)
}
