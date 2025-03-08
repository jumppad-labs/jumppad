package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/gosuri/uitable/util/strutil"
	"github.com/stretchr/testify/require"
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

	require.True(t, is)
}

func TestIsFolderNotExists(t *testing.T) {
	is := IsLocalFolder("/dfdfdf")

	require.False(t, is)
}

func TestIsNotFolder(t *testing.T) {
	is := IsLocalFolder("github.com/")

	require.False(t, is)
}

func TestGetBlueprintFolderReturnsFolder(t *testing.T) {
	dir, err := BlueprintFolder("github.com/org/repo?ref=dfdf&foo=bah//folder")

	require.NoError(t, err)
	require.Equal(t, "folder", dir)
}

func TestGetBlueprintFolderReturnsError(t *testing.T) {
	_, err := BlueprintFolder("github.com/org/repo/folder")

	require.Error(t, err)
}

func TestValidatesNameCorrectly(t *testing.T) {
	ok, err := ValidateName("abc-sdf")
	require.NoError(t, err)
	require.True(t, ok)
}

func TestValidatesNameAndReturnsErrorWhenInvalid(t *testing.T) {
	ok, err := ValidateName("*$-abcd")
	require.Error(t, err)
	require.False(t, ok)
}

func TestValidatesNameAndReturnsErrorWhenTooLong(t *testing.T) {
	dn := strutil.PadLeft("a", 129, 'a')

	ok, err := ValidateName(dn)

	require.Error(t, err)
	require.False(t, ok)
}

func TestFQDNReturnsCorrectValue(t *testing.T) {
	fq := FQDN("test", "", "type")
	require.Equal(t, "test.type.local.jmpd.in", fq)
}

func TestFQDNReplacesInvalidChars(t *testing.T) {
	fq := FQDN("tes&t", "", "kubernetes_cluster")
	require.Equal(t, "tes-t.kubernetes-cluster.local.jmpd.in", fq)
}

func TestFQDNVolumeReturnsCorrectValue(t *testing.T) {
	fq := FQDNVolumeName("test")
	require.Equal(t, "test.volume.jmpd.in", fq)
}

func TestHomeReturnsCorrectValue(t *testing.T) {
	h := HomeFolder()
	require.Equal(t, os.Getenv(HomeEnvName()), h)
}

func TestStateReturnsCorrectValue(t *testing.T) {
	h := StateDir()
	expected := filepath.Join(os.Getenv(HomeEnvName()), ".jumppad/state")

	require.Equal(t, expected, h)
}

func TestStatePathReturnsCorrectValue(t *testing.T) {
	h := StatePath()
	require.Equal(t, filepath.Join(os.Getenv(HomeEnvName()), ".jumppad/state/state.json"), h)
}

func TestCreateKubeConfigPathReturnsCorrectValues(t *testing.T) {
	home := os.Getenv(HomeEnvName())
	tmp, _ := os.MkdirTemp("", "")
	os.Setenv(HomeEnvName(), tmp)
	defer os.Setenv(HomeEnvName(), home)

	d, f, dp := CreateKubeConfigPath("testing")

	require.Equal(t, filepath.Join(tmp, ".jumppad", "config", "testing"), d)
	require.Equal(t, filepath.Join(tmp, ".jumppad", "config", "testing", "kubeconfig.yaml"), f)
	require.Equal(t, filepath.Join(tmp, ".jumppad", "config", "testing", "kubeconfig-docker.yaml"), dp)

	// check creates folder
	s, err := os.Stat(d)
	require.NoError(t, err)
	require.True(t, s.IsDir())
}

func TestShipyardTempReturnsPath(t *testing.T) {
	home := os.Getenv(HomeEnvName())
	tmp, _ := os.MkdirTemp("", "")
	os.Setenv(HomeEnvName(), tmp)

	t.Cleanup(func() {
		os.Setenv(HomeEnvName(), home)
		os.RemoveAll(tmp)
	})

	st := JumppadTemp()

	require.Equal(t, filepath.Join(tmp, ".jumppad", "/tmp"), st)

	s, err := os.Stat(st)
	require.NoError(t, err)
	require.True(t, s.IsDir())
}

func TestShipyardDataReturnsPath(t *testing.T) {
	home := os.Getenv(HomeEnvName())
	tmp, _ := os.MkdirTemp("", "")
	os.Setenv(HomeEnvName(), tmp)

	t.Cleanup(func() {
		os.Setenv(HomeEnvName(), home)
		os.RemoveAll(tmp)
	})

	d := DataFolder("test", 0775)

	require.Equal(t, filepath.Join(tmp, ".jumppad", "/data", "/test"), d)

	s, err := os.Stat(d)
	fmt.Println(d, s)

	require.NoError(t, err)
	require.True(t, s.IsDir())
}

func TestHelmLocalFolderReturnsPath(t *testing.T) {
	chart := "github.com/jetstack/cert-manager?ref=v1.2.0/deploy/charts//cert-manager"
	h := HelmLocalFolder(chart)

	require.Equal(t, filepath.Join(os.Getenv(HomeEnvName()), ".jumppad", "/helm_charts", "github.com/jetstack/cert-manager/ref-v1.2.0/deploy/charts/cert-manager"), h)
}

func TestShipyardReleasesReturnsPath(t *testing.T) {
	r := ReleasesFolder()

	require.Equal(t, filepath.Join(os.Getenv(HomeEnvName()), ".jumppad", "/releases"), r)
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
	dst := BlueprintLocalFolder("github.com/shipyard-run/blueprints?ref=dfdf&foo=bah//vault-k8s")

	require.Equal(t, filepath.Join(JumppadHome(), "/blueprints/github.com/shipyard-run/blueprints/ref-dfdf-foo-bah/vault-k8s"), dst)
}

func TestDockerHostWithDefaultReturnsCorrectValue(t *testing.T) {
	dh := os.Getenv("DOCKER_HOST")
	os.Unsetenv("DOCKER_HOST")
	t.Cleanup(func() {
		os.Setenv("DOCKER_HOST", dh)
	})

	ds := GetDockerHost()
	require.Equal(t, "/var/run/docker.sock", ds)
}

func TestGetLocalIPAndHostnameReturnsCorrectly(t *testing.T) {
	ip, host := GetLocalIPAndHostname()

	require.NotEqual(t, ip, "127.0.0.1")
	require.NotEqual(t, host, "localhost")
}

func TestImageCacheAddressReturnsDefaultWhenEnvNotSet(t *testing.T) {
	proxy := ImageCacheAddress()

	require.Equal(t, jumppadProxyAddress, proxy)
}

func TestImageCacheAddressReturnsEnvWhenEnvSet(t *testing.T) {
	httpProxy := "http://myproxy.com"
	os.Setenv("IMAGE_CACHE_ADDR", httpProxy)
	proxy := ImageCacheAddress()

	require.Equal(t, httpProxy, proxy)
}

var testData = `
{
	"checks": "test",
	"children": [
		{
			"checks": "test"
		},
		{
			"checks": "test2"
		}
	]
}
`

func TestChecksumInterface(t *testing.T) {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(testData), &data)
	require.NoError(t, err)

	c, err := ChecksumFromInterface(data)
	require.NoError(t, err)

	require.Equal(t, "h1:kpp5xuYieKQMhbtP0+Y6N+dUzx9p9pGq9+WXkgbK6fs=", c)
}
