package resources

import (
	"os"
	"path"
	"testing"

	"github.com/shipyard-run/hclconfig/types"
	"github.com/stretchr/testify/require"
)

func TestNomadJobProcessSetsAbsolute(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	c := &NomadJob{
		ResourceMetadata: types.ResourceMetadata{File: "./"},
		Paths: []string{
			"./one.hcl",
			"./two.hcl",
		},
	}

	c.Process()

	require.Equal(t, path.Join(wd, "one.hcl"), c.Paths[0])
	require.Equal(t, path.Join(wd, "two.hcl"), c.Paths[1])
}
