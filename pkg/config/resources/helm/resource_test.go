package helm

import (
	"os"
	"path"
	"testing"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/stretchr/testify/require"
)

func TestHelmProcessSetsAbsolute(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	h := &Helm{
		ResourceMetadata: types.ResourceMetadata{File: "./"},
		Chart:            "./",
		Values:           "./values.yaml",
	}

	err = h.Process()
	require.NoError(t, err)

	require.Equal(t, wd, h.Chart)
	require.Equal(t, path.Join(wd, "values.yaml"), h.Values)
}
