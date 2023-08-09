package exec

import (
	"os"
	"testing"

	"github.com/jumppad-labs/hclconfig/types"
	ctypes "github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	"github.com/stretchr/testify/require"
)

func TestRemoteExecProcessSetsAbsolute(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	c := &RemoteExec{
		ResourceMetadata: types.ResourceMetadata{File: "./"},
		Volumes: []ctypes.Volume{
			{
				Source:      "./",
				Destination: "./",
			},
		},
	}

	c.Process()

	require.Equal(t, wd, c.Volumes[0].Source)
}
