package terraform

import (
	"os"
	"path"
	"testing"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/container/mocks"
	ctypes "github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/container"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

func setupProvider(t *testing.T, c *Terraform) (*TerraformProvider, *mocks.ContainerTasks, string) {
	// set the home directory to a temporary directory so that everthing is cleaned up
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)

	// output should always exist
	sd := terraformStateFolder(c)
	os.WriteFile(path.Join(sd, "output.json"), []byte("{\"abc\": {\"value\": \"123\"}}"), 0655)

	mc := &mocks.ContainerTasks{}
	mc.Mock.On("PullImage", mock.Anything, false).Return(nil)
	mc.Mock.On("CreateContainer", mock.Anything).Return("abc", nil)
	mc.Mock.On("ExecuteScript", "abc", mock.Anything, mock.Anything, mock.Anything, "root", mock.Anything, 300, mock.Anything).Return(0, nil)
	mc.Mock.On("RemoveContainer", "abc", true).Return(nil)

	l := logger.NewTestLogger(t)
	p := &TerraformProvider{c, mc, l}

	return p, mc, sd
}

func TestCreateWithNoVariablesDoesNotReturnError(t *testing.T) {
	p, _, _ := setupProvider(t, &Terraform{ResourceMetadata: types.ResourceMetadata{Name: "test"}})

	err := p.Create()
	require.NoError(t, err)
}

func TestCreateWithWithVariablesGeneratesFile(t *testing.T) {
	variables := cty.MapVal(map[string]cty.Value{
		"foo": cty.StringVal("bar"),
	})

	p, _, sd := setupProvider(t, &Terraform{
		ResourceMetadata: types.ResourceMetadata{Name: "test"},
		Variables:        variables,
	},
	)

	err := p.Create()
	require.NoError(t, err)

	// check variables file
	d, err := os.ReadFile(path.Join(sd, "terraform.tfvars"))
	require.NoError(t, err)
	require.Equal(t, "foo = \"bar\"\n", string(d))
}

func TestCreatesTerraformContainerWithTheCorrectValues(t *testing.T) {
	res := &Terraform{
		ResourceMetadata: types.ResourceMetadata{Name: "test"},
		Networks: []container.NetworkAttachment{
			container.NetworkAttachment{
				ID: "Abc123",
			},
		},
		Version: "1.16.2",
		Source:  "../../../../examples/terraform/workspace",
	}

	p, m, _ := setupProvider(t, res)

	err := p.Create()
	require.NoError(t, err)

	c := m.Calls[1].Arguments[0].(*ctypes.Container)

	// ensure the terraform plugin cache is added so that plugins are not constantly downloaded
	require.Equal(t, c.Environment["TF_PLUGIN_CACHE_DIR"], "/var/lib/terraform.d")

	// check the correct image is used
	require.Equal(t, c.Image.Name, "hashicorp/terraform:1.16.2")

	// check the networks have been added
	require.Equal(t, c.Networks[0].ID, "Abc123")

	// check the convig has been added
	require.Equal(t, "../../../../examples/terraform/workspace", c.Volumes[0].Source)
	require.Equal(t, "/config", c.Volumes[0].Destination)

	// check the state volume has been added
	require.Equal(t, terraformStateFolder(res), c.Volumes[1].Source)
	require.Equal(t, "/var/lib/terraform", c.Volumes[1].Destination)

	// check the plugin cache has been added
	require.Equal(t, terraformCacheFolder(), c.Volumes[2].Source)
	require.Equal(t, "/var/lib/terraform.d", c.Volumes[2].Destination)
}

func TestCreateExecutesCommandInContainer(t *testing.T) {
	res := &Terraform{
		ResourceMetadata: types.ResourceMetadata{Name: "test"},
		Networks: []container.NetworkAttachment{
			container.NetworkAttachment{
				ID: "Abc123",
			},
		},
		WorkingDirectory: "/test",
	}

	p, m, _ := setupProvider(t, res)

	err := p.Create()
	require.NoError(t, err)

	// assert script was executed in container
	script := m.Calls[2].Arguments[1].(string)

	require.Contains(t, script, "terraform init")
	require.Contains(t, script, "terraform apply")
	require.Contains(t, script, "terraform output")

	// check the working directory is set
	wd := m.Calls[2].Arguments[3].(string)
	require.Equal(t, "/config/test", wd)

	// ensure that the .terraform directory is removed
	// this contains the providers but these are symlinks
	// from the cache
	// assert cleanup was executed in container
	script = m.Calls[3].Arguments[1].(string)
	require.Contains(t, script, "rm -rf /config/")
}

func TestCreateSetsOutput(t *testing.T) {
	res := &Terraform{
		ResourceMetadata: types.ResourceMetadata{Name: "test"},
		Networks: []container.NetworkAttachment{
			container.NetworkAttachment{
				ID: "Abc123",
			},
		},
		WorkingDirectory: "/test",
	}

	p, _, _ := setupProvider(t, res)

	err := p.Create()
	require.NoError(t, err)

	require.Equal(t, "123", res.Output.AsValueMap()["abc"].AsString())
}

func TestDestroyExecutesCommandInContainer(t *testing.T) {
	res := &Terraform{
		ResourceMetadata: types.ResourceMetadata{Name: "test"},
		Networks: []container.NetworkAttachment{
			container.NetworkAttachment{
				ID: "Abc123",
			},
		},
		WorkingDirectory: "/test",
	}

	p, m, sd := setupProvider(t, res)

	// setup fake state or destroy will not do anything
	os.WriteFile(path.Join(sd, "terraform.tfstate"), []byte("{\"abc\": {\"value\": \"123\"}}"), 0655)
	os.WriteFile(path.Join(sd, "terraform.tfvars"), []byte("{\"abc\": {\"value\": \"123\"}}"), 0655)

	err := p.Destroy()
	require.NoError(t, err)

	// assert script was executed in container
	script := m.Calls[2].Arguments[1].(string)

	require.Contains(t, script, "terraform init")
	require.Contains(t, script, "terraform destroy")

	// check the working directory is set
	wd := m.Calls[2].Arguments[3].(string)
	require.Equal(t, "/config/test", wd)
}
