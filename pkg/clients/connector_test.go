package clients

import (
	"fmt"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"

	"github.com/shipyard-run/shipyard/pkg/utils"
	assert "github.com/stretchr/testify/require"
)

var suiteTemp string
var suiteBinary string
var suiteCertBundle *CertBundle
var suiteOptions ConnectorOptions

func TestConnectorSuite(t *testing.T) {
	suiteTemp = t.TempDir()
	suiteBinary = utils.GetShipyardBinaryPath()

	suiteOptions.LogDirectory = os.TempDir()
	suiteOptions.BinaryPath = suiteBinary
	suiteOptions.GrpcBind = fmt.Sprintf(":%d", rand.Intn(1000)+20000)
	suiteOptions.HTTPBind = fmt.Sprintf(":%d", rand.Intn(1000)+20000)

	t.Run("Generates certificates", testGenerateCreatesBundle)
	t.Run("Fetches certificates", testFetchesLocalCertBundle)
	t.Run("Generates a leaf certificate", testGenerateCreatesLeaf)
	t.Run("Starts Connector correctly", testStartsConnector)
}

func testGenerateCreatesBundle(t *testing.T) {
	c := NewConnector(suiteOptions)

	var err error
	suiteCertBundle, err = c.GenerateLocalCertBundle(suiteTemp)
	assert.NoError(t, err)

	assert.FileExists(t, suiteCertBundle.RootCertPath)
	assert.FileExists(t, suiteCertBundle.RootKeyPath)
	assert.FileExists(t, suiteCertBundle.LeafKeyPath)
	assert.FileExists(t, suiteCertBundle.LeafCertPath)
}

func testFetchesLocalCertBundle(t *testing.T) {
	c := NewConnector(suiteOptions)

	cb, err := c.GetLocalCertBundle(suiteTemp)
	assert.NoError(t, err)
	assert.NotNil(t, cb)
}

func testGenerateCreatesLeaf(t *testing.T) {
	c := NewConnector(suiteOptions)

	certDir := path.Join(suiteTemp, "tester")
	os.MkdirAll(certDir, os.ModePerm)

	var err error
	leafCertBundle, err := c.GenerateLeafCert(
		suiteCertBundle.RootKeyPath,
		suiteCertBundle.RootCertPath,
		[]string{"tester"},
		[]string{"123.121.121.1"},
		certDir,
	)
	assert.NoError(t, err)

	assert.FileExists(t, leafCertBundle.RootCertPath)
	assert.FileExists(t, leafCertBundle.RootKeyPath)
	assert.FileExists(t, leafCertBundle.LeafKeyPath)
	assert.FileExists(t, leafCertBundle.LeafCertPath)
}

func testStartsConnector(t *testing.T) {
	c := NewConnector(suiteOptions)

	err := c.Start(suiteCertBundle)
	assert.NoError(t, err)

	// make sure we stop even if we fail
	defer c.Stop()

	// check the logfile
	assert.FileExists(t, path.Join(os.TempDir(), "connector.log"))

	// check is running
	assert.Eventually(t, func() bool {
		return c.IsRunning()
	}, 3*time.Second, 100*time.Millisecond)
}
