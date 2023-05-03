package cmd

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/jumppad-labs/connector/crypto"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupGenerateCerts(t *testing.T, dir string) {
	// create a tempfolder
	k, err := crypto.GenerateKeyPair()
	assert.NoError(t, err)

	ca, err := crypto.GenerateCA("CA", k.Private)
	assert.NoError(t, err)

	err = k.Private.WriteFile(filepath.Join(dir, "root.key"))
	assert.NoError(t, err)

	err = ca.WriteFile(filepath.Join(dir, "root.cert"))
	assert.NoError(t, err)

	lk, err := crypto.GenerateKeyPair()
	assert.NoError(t, err)

	lc, err := crypto.GenerateLeaf(
		"Leaf",
		[]string{"127.0.0.1"},
		[]string{"localhost"},
		ca,
		k.Private,
		lk.Private,
	)

	err = lk.Private.WriteFile(filepath.Join(dir, "leaf.key"))
	assert.NoError(t, err)

	err = lc.WriteFile(filepath.Join(dir, "leaf.cert"))
	assert.NoError(t, err)
}

func setupServerRunTests(t *testing.T) string {
	// setup the certificates
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)

	oh := os.Getenv(utils.HomeEnvName())
	os.Setenv(utils.HomeEnvName(), tmpDir)

	setupGenerateCerts(t, utils.CertsDir(""))

	t.Cleanup(func() {
		os.Setenv(utils.HomeEnvName(), oh)
		os.RemoveAll(tmpDir)
	})

	return utils.CertsDir("")
}

func TestServerStarts(t *testing.T) {
	cd := setupServerRunTests(t)

	grpcPort := rand.Intn(1000) + 10000
	httpPort := rand.Intn(1000) + 10000
	crc := newConnectorRunCommand()
	crc.Flags().Set("grpc-bind", fmt.Sprintf("127.0.0.1:%d", grpcPort))
	crc.Flags().Set("http-bind", fmt.Sprintf("127.0.0.1:%d", httpPort))
	crc.Flags().Set("root-cert-path", path.Join(cd, "/root.cert"))
	crc.Flags().Set("server-cert-path", path.Join(cd, "/leaf.cert"))
	crc.Flags().Set("server-key-path", path.Join(cd, "/leaf.key"))

	go crc.Execute()

	// test we can call the http health endpoint
	require.Eventually(t, func() bool {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		resp, err := http.DefaultClient.Get(fmt.Sprintf("https://127.0.0.1:%d/health", httpPort))
		if err != nil || resp.StatusCode != 200 {
			return false
		}

		return true
	}, 10*time.Second, 100*time.Millisecond)
}

func TestServerErrorBadServerCerts(t *testing.T) {
	cd := setupServerRunTests(t)

	grpcPort := rand.Intn(1000) + 10000
	httpPort := rand.Intn(1000) + 10000
	crc := newConnectorRunCommand()
	crc.Flags().Set("grpc-bind", fmt.Sprintf("127.0.0.1:%d", grpcPort))
	crc.Flags().Set("http-bind", fmt.Sprintf("127.0.0.1:%d", httpPort))
	crc.Flags().Set("root-cert-path", path.Join(cd, "/root.cert"))
	crc.Flags().Set("server-cert-path", path.Join(cd, "/leaf.bad"))
	crc.Flags().Set("server-key-path", path.Join(cd, "/leaf.bad"))

	err := crc.Execute()
	require.Error(t, err)
}

func TestServerErrorBadServerCA(t *testing.T) {
	cd := setupServerRunTests(t)

	grpcPort := rand.Intn(1000) + 10000
	httpPort := rand.Intn(1000) + 10000
	crc := newConnectorRunCommand()
	crc.Flags().Set("grpc-bind", fmt.Sprintf("127.0.0.1:%d", grpcPort))
	crc.Flags().Set("http-bind", fmt.Sprintf("127.0.0.1:%d", httpPort))
	crc.Flags().Set("root-cert-path", path.Join(cd, "/root.bad"))
	crc.Flags().Set("server-cert-path", path.Join(cd, "/leaf.cert"))
	crc.Flags().Set("server-key-path", path.Join(cd, "/leaf.key"))

	err := crc.Execute()
	require.Error(t, err)
}

func TestServerLogsToFile(t *testing.T) {
	cd := setupServerRunTests(t)

	grpcPort := rand.Intn(1000) + 10000
	httpPort := rand.Intn(1000) + 10000
	crc := newConnectorRunCommand()
	crc.Flags().Set("grpc-bind", fmt.Sprintf("127.0.0.1:%d", grpcPort))
	crc.Flags().Set("http-bind", fmt.Sprintf("127.0.0.1:%d", httpPort))
	crc.Flags().Set("root-cert-path", path.Join(cd, "/root.cert"))
	crc.Flags().Set("server-cert-path", path.Join(cd, "/leaf.cert"))
	crc.Flags().Set("server-key-path", path.Join(cd, "/leaf.key"))
	crc.Flags().Set("log-file", utils.GetConnectorLogFile())

	go crc.Execute()

	require.Eventually(t, func() bool {
		if _, err := os.Stat(utils.GetConnectorLogFile()); err != nil {
			return false
		}

		return true
	}, 10*time.Second, 100*time.Millisecond)
}
