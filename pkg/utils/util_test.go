package utils

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/gosuri/uitable/util/strutil"
	"github.com/stretchr/testify/assert"
)

func TestIsLocalFolder(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		// TODO: Add test cases.
		{
			"False when directory not exist",
			"/tmpsfsfsd",
			false,
		}, {
			"True when current directory",
			"./",
			true,
		}, {
			"True when previous directory",
			"../",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsLocalFolder(tt.path); got != tt.want {
				t.Errorf("IsLocalFolder() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsLocalAbsFolder(t *testing.T) {
	is := IsLocalFolder("/tmp")

	assert.True(t, is)
}

func TestIsFolderNotExists(t *testing.T) {
	is := IsLocalFolder("/dfdfdf")

	assert.False(t, is)
}

func TestIsNotFolder(t *testing.T) {
	is := IsLocalFolder("github.com/")

	assert.False(t, is)
}

func TestGetBlueprintFolderReturnsFolder(t *testing.T) {
	dir, err := GetBlueprintFolder("github.com/org/repo//folder?ref=dfdf&foo=bah")

	assert.NoError(t, err)
	assert.Equal(t, "folder/ref/dfdf/foo/bah", dir)
}

func TestGetBlueprintFolderReturnsError(t *testing.T) {
	_, err := GetBlueprintFolder("github.com/org/repo/folder")

	assert.Error(t, err)
}

func TestValidatesNameCorrectly(t *testing.T) {
	ok, err := ValidateName("abc-sdf")
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestValidatesNameAndReturnsErrorWhenInvalid(t *testing.T) {
	ok, err := ValidateName("*$-abcd")
	assert.Error(t, err)
	assert.False(t, ok)
}

func TestValidatesNameAndReturnsErrorWhenTooLong(t *testing.T) {
	dn := strutil.PadLeft("a", 129, 'a')

	ok, err := ValidateName(dn)

	assert.Error(t, err)
	assert.False(t, ok)
}

func TestFQDNReturnsCorrectValue(t *testing.T) {
	fq := FQDN("test", "type")
	assert.Equal(t, "test.type.shipyard.run", fq)
}

func TestFQDNReplacesInvalidChars(t *testing.T) {
	fq := FQDN("tes&t", "type")
	assert.Equal(t, "tes-t.type.shipyard.run", fq)
}

func TestFQDNVolumeReturnsCorrectValue(t *testing.T) {
	fq := FQDNVolumeName("test")
	assert.Equal(t, "test.volume.shipyard.run", fq)
}

func TestHomeReturnsCorrectValue(t *testing.T) {
	h := HomeFolder()
	assert.Equal(t, os.Getenv(HomeEnvName()), h)
}

func TestStateReturnsCorrectValue(t *testing.T) {
	h := StateDir()
	expected := filepath.Join(os.Getenv(HomeEnvName()), ".shipyard/state")

	assert.Equal(t, expected, h)
}

func TestStatePathReturnsCorrectValue(t *testing.T) {
	h := StatePath()
	assert.Equal(t, filepath.Join(os.Getenv(HomeEnvName()), ".shipyard/state/state.json"), h)
}

func TestCreateKubeConfigPathReturnsCorrectValues(t *testing.T) {
	home := os.Getenv(HomeEnvName())
	tmp, _ := ioutil.TempDir("", "")
	os.Setenv(HomeEnvName(), tmp)
	defer os.Setenv(HomeEnvName(), home)

	d, f, dp := CreateKubeConfigPath("testing")

	assert.Equal(t, filepath.Join(tmp, ".shipyard", "config", "testing"), d)
	assert.Equal(t, filepath.Join(tmp, ".shipyard", "config", "testing", "kubeconfig.yaml"), f)
	assert.Equal(t, filepath.Join(tmp, ".shipyard", "config", "testing", "kubeconfig-docker.yaml"), dp)

	// check creates folder
	s, err := os.Stat(d)
	assert.NoError(t, err)
	assert.True(t, s.IsDir())
}

func TestCreateClusterConfigPathReturnsCorrectValues(t *testing.T) {
	home := os.Getenv(HomeEnvName())
	tmp, _ := ioutil.TempDir("", "")
	os.Setenv(HomeEnvName(), tmp)
	defer os.Setenv(HomeEnvName(), home)

	d, f := CreateClusterConfigPath("testing")

	assert.Equal(t, filepath.Join(tmp, ".shipyard", "config", "testing"), d)
	assert.Equal(t, filepath.Join(tmp, ".shipyard", "config", "testing", "config.json"), f)

	// check creates folder
	s, err := os.Stat(d)
	assert.NoError(t, err)
	assert.True(t, s.IsDir())
}

func TestShipyardTempReturnsPath(t *testing.T) {
	home := os.Getenv(HomeEnvName())
	tmp, _ := ioutil.TempDir("", "")
	os.Setenv(HomeEnvName(), tmp)

	t.Cleanup(func() {
		os.Setenv(HomeEnvName(), home)
		os.RemoveAll(tmp)
	})

	st := ShipyardTemp()

	assert.Equal(t, filepath.Join(tmp, ".shipyard", "/tmp"), st)

	s, err := os.Stat(st)
	assert.NoError(t, err)
	assert.True(t, s.IsDir())
}

func TestShipyardDataReturnsPath(t *testing.T) {
	home := os.Getenv(HomeEnvName())
	tmp, _ := ioutil.TempDir("", "")
	os.Setenv(HomeEnvName(), tmp)

	t.Cleanup(func() {
		os.Setenv(HomeEnvName(), home)
		os.RemoveAll(tmp)
	})

	d := GetDataFolder("test")

	assert.Equal(t, filepath.Join(tmp, ".shipyard", "/data", "/test"), d)

	s, err := os.Stat(d)
	assert.NoError(t, err)
	assert.True(t, s.IsDir())
}

func TestShipyardHelmReturnsPath(t *testing.T) {
	h := GetHelmLocalFolder("test")

	assert.Equal(t, filepath.Join(os.Getenv(HomeEnvName()), ".shipyard", "/helm_charts", "/test"), h)
}

func TestShipyardReleasesReturnsPath(t *testing.T) {
	r := GetReleasesFolder()

	assert.Equal(t, filepath.Join(os.Getenv(HomeEnvName()), ".shipyard", "/releases"), r)
}

func TestIsHCLFile(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		// TODO: Add test cases.
		{
			"False when directory not exist",
			"/tmpsfsfsd",
			false,
		}, {
			"False when directory",
			"/tmp",
			false,
		}, {
			"True when .hcl file",
			"../../examples/single_k3s_cluster/k8s.hcl",
			true,
		}, {
			"False when other file",
			"../../examples/single_k3s_cluster/helm/consul-values.yaml",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsHCLFile(tt.path); got != tt.want {
				t.Errorf("IsHCLFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlueprintLocalFolder(t *testing.T) {
	dst := GetBlueprintLocalFolder("github.com/shipyard-run/blueprints//vault-k8s?ref=dfdf&foo=bah")

	assert.Equal(t, filepath.Join(ShipyardHome(), "/blueprints/github.com/shipyard-run/blueprints/vault-k8s/ref/dfdf/foo/bah"), dst)
}

func TestDockerHostWithDefaultReturnsCorrectValue(t *testing.T) {
	dh := os.Getenv("DOCKER_HOST")
	os.Unsetenv("DOCKER_HOST")
	t.Cleanup(func() {
		os.Setenv("DOCKER_HOST", dh)
	})

	ds := GetDockerHost()
	assert.Equal(t, "/var/run/docker.sock", ds)
}
