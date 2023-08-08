package resources

import (
	"os"
	"path"
	"testing"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/stretchr/testify/require"
)

func TestK8sConfigProcessSetsAbsolute(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	k := &K8sConfig{
		ResourceMetadata: types.ResourceMetadata{File: "./"},
		Paths:            []string{"./one.yaml", "./two.yaml"},
	}

	err = k.Process()
	require.NoError(t, err)

	require.Equal(t, path.Join(wd, "one.yaml"), k.Paths[0])
	require.Equal(t, path.Join(wd, "two.yaml"), k.Paths[1])
}
