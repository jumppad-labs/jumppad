package cmd

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path"
	"testing"
	"time"

	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/stretchr/testify/require"
)

func setupServerRunTests(t *testing.T) string {
	// setup the certificates
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)

	oh := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)

	// create the shipyard home folder
	os.MkdirAll(utils.ShipyardHome(), 0777)

	// generate the CA certs
	ccc := newConnectorCertCmd()
	ccc.Flags().Set("ca", "true") // generate a ca
	ccc.SetArgs([]string{tmpDir}) // set the output directory
	err = ccc.Execute()
	require.NoError(t, err)

	// generate the leaf
	ccc = newConnectorCertCmd()
	ccc.Flags().Set("leaf", "true") // generate a leaf
	ccc.Flags().Set("root-key", path.Join(tmpDir, "root.key"))
	ccc.Flags().Set("root-ca", path.Join(tmpDir, "root.cert"))
	ccc.SetArgs([]string{tmpDir}) // set the output directory
	err = ccc.Execute()
	require.NoError(t, err)

	//err := connectorRunCmd.Execute()
	//require.NoError(t, err)

	t.Cleanup(func() {
		os.Setenv("HOME", oh)
		os.RemoveAll(tmpDir)
	})

	return tmpDir
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
