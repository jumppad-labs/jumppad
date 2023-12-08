package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/gosuri/uitable/util/strutil"
	"github.com/stretchr/testify/require"
	assert "github.com/stretchr/testify/require"
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
	dir, err := BlueprintFolder("github.com/org/repo?ref=dfdf&foo=bah//folder")

	assert.NoError(t, err)
	assert.Equal(t, "folder", dir)
}

func TestGetBlueprintFolderReturnsError(t *testing.T) {
	_, err := BlueprintFolder("github.com/org/repo/folder")

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
	fq := FQDN("test", "", "type")
	assert.Equal(t, "test.type.jumppad.dev", fq)
}

func TestFQDNReplacesInvalidChars(t *testing.T) {
	fq := FQDN("tes&t", "", "kubernetes_cluster")
	assert.Equal(t, "tes-t.kubernetes-cluster.jumppad.dev", fq)
}

func TestFQDNVolumeReturnsCorrectValue(t *testing.T) {
	fq := FQDNVolumeName("test")
	assert.Equal(t, "test.volume.jumppad.dev", fq)
}

func TestHomeReturnsCorrectValue(t *testing.T) {
	h := HomeFolder()
	assert.Equal(t, os.Getenv(HomeEnvName()), h)
}

func TestStateReturnsCorrectValue(t *testing.T) {
	h := StateDir()
	expected := filepath.Join(os.Getenv(HomeEnvName()), ".jumppad/state")

	assert.Equal(t, expected, h)
}

func TestStatePathReturnsCorrectValue(t *testing.T) {
	h := StatePath()
	assert.Equal(t, filepath.Join(os.Getenv(HomeEnvName()), ".jumppad/state/state.json"), h)
}

func TestCreateKubeConfigPathReturnsCorrectValues(t *testing.T) {
	home := os.Getenv(HomeEnvName())
	tmp, _ := ioutil.TempDir("", "")
	os.Setenv(HomeEnvName(), tmp)
	defer os.Setenv(HomeEnvName(), home)

	d, f, dp := CreateKubeConfigPath("testing")

	assert.Equal(t, filepath.Join(tmp, ".jumppad", "config", "testing"), d)
	assert.Equal(t, filepath.Join(tmp, ".jumppad", "config", "testing", "kubeconfig.yaml"), f)
	assert.Equal(t, filepath.Join(tmp, ".jumppad", "config", "testing", "kubeconfig-docker.yaml"), dp)

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

	st := JumppadTemp()

	assert.Equal(t, filepath.Join(tmp, ".jumppad", "/tmp"), st)

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

	d := DataFolder("test", 0775)

	assert.Equal(t, filepath.Join(tmp, ".jumppad", "/data", "/test"), d)

	s, err := os.Stat(d)
	fmt.Println(d, s)

	assert.NoError(t, err)
	assert.True(t, s.IsDir())
}

func TestHelmLocalFolderReturnsPath(t *testing.T) {
	chart := "github.com/jetstack/cert-manager?ref=v1.2.0/deploy/charts//cert-manager"
	h := HelmLocalFolder(chart)

	assert.Equal(t, filepath.Join(os.Getenv(HomeEnvName()), ".jumppad", "/helm_charts", "github.com/jetstack/cert-manager/ref-v1.2.0/deploy/charts/cert-manager"), h)
}

func TestShipyardReleasesReturnsPath(t *testing.T) {
	r := ReleasesFolder()

	assert.Equal(t, filepath.Join(os.Getenv(HomeEnvName()), ".jumppad", "/releases"), r)
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

	assert.Equal(t, filepath.Join(JumppadHome(), "/blueprints/github.com/shipyard-run/blueprints/ref-dfdf-foo-bah/vault-k8s"), dst)
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

func TestGetLocalIPAndHostnameReturnsCorrectly(t *testing.T) {
	ip, host := GetLocalIPAndHostname()

	assert.NotEqual(t, ip, "127.0.0.1")
	assert.NotEqual(t, host, "localhost")
}

func TestHTTPProxyAddressReturnsDefaultWhenEnvNotSet(t *testing.T) {
	proxy := HTTPProxyAddress()

	assert.Equal(t, jumppadProxyAddress, proxy)
}

func TestHTTPSProxyAddressReturnsDefaultWhenEnvNotSet(t *testing.T) {
	proxy := HTTPSProxyAddress()

	assert.Equal(t, jumppadProxyAddress, proxy)
}

func TestHTTPProxyAddressReturnsEnvWhenEnvSet(t *testing.T) {
	httpProxy := "http://myproxy.com"
	os.Setenv("HTTP_PROXY", httpProxy)
	proxy := HTTPProxyAddress()

	assert.Equal(t, httpProxy, proxy)
}

func TestHTTPSProxyAddressReturnsEnvWhenEnvSet(t *testing.T) {
	httpsProxy := "https://myproxy.com"
	os.Setenv("HTTPS_PROXY", httpsProxy)
	proxy := HTTPSProxyAddress()

	assert.Equal(t, httpsProxy, proxy)
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

	assert.Equal(t, "h1:kpp5xuYieKQMhbtP0+Y6N+dUzx9p9pGq9+WXkgbK6fs=", c)
}
