package nomad

import (
	"os"
	"path"
	"testing"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/stretchr/testify/require"
)

func TestNomadJobProcessSetsAbsolute(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	c := &NomadJob{
		ResourceBase: types.ResourceBase{Meta: types.Meta{File: "./"}},
		Paths: []string{
			"./one.hcl",
			"./two.hcl",
		},
	}

	c.Process()

	require.Equal(t, path.Join(wd, "one.hcl"), c.Paths[0])
	require.Equal(t, path.Join(wd, "two.hcl"), c.Paths[1])
}
