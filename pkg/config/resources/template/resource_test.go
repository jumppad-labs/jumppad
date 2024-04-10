package template

import (
	"os"
	"path"
	"testing"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/stretchr/testify/require"
)

func TestTemplateProcessSetsAbsoluteWhenBothFiles(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	c := &Template{
		ResourceBase: types.ResourceBase{Meta: types.Meta{File: "./"}},
		Source:       "./",
		Destination:  "./output.hcl",
	}

	c.Process()

	require.Equal(t, path.Join(wd, "output.hcl"), c.Destination)
	require.Equal(t, wd, c.Source)
}

func TestTemplateProcessSetsAbsoluteWhenSourceString(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	c := &Template{
		ResourceBase: types.ResourceBase{Meta: types.Meta{File: "./"}},
		Source:       "foobar",
		Destination:  "./output.hcl",
	}

	c.Process()

	require.Equal(t, path.Join(wd, "output.hcl"), c.Destination)
	require.Equal(t, "foobar", c.Source)
}
