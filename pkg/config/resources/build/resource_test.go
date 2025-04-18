package build

import (
	"testing"

	"github.com/instruqt/jumppad/pkg/config"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/stretchr/testify/require"
)

func init() {
	config.RegisterResource(TypeBuild, &Build{}, &Provider{})
}

func TestBuildRaisesErrorWhenDockerfileOutsideContext(t *testing.T) {
	c := &Build{
		ResourceBase: types.ResourceBase{Meta: types.Meta{File: "./"}},
		Container: BuildContainer{
			Context:    "../../../../examples/build/src",
			DockerFile: "/Dockerfile/Dockerfile",
		},
	}

	err := c.Process()
	require.Error(t, err)
}

func TestBuildNoErrorWhenDockerfileInContext(t *testing.T) {
	c := &Build{
		ResourceBase: types.ResourceBase{Meta: types.Meta{File: "./"}},
		Container: BuildContainer{
			Context:    "../../../../examples/build/src",
			DockerFile: "./Docker/Dockerfile",
		},
	}

	err := c.Process()
	require.NoError(t, err)
}
