package clients

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"
)

func buildConnector(t *testing.T) string {
	// we need a shipyard binary to run for connector tests
	// build a binary
	args := []string{}

	_, filename, _, _ := runtime.Caller(0)
	dir := path.Dir(filename)

	fp := ""

	parent := 0
	// walk backwards until we find the go.mod
	for {
		if parent == 5 {
			t.Fatal("Unable to find source directory")
		}

		files, err := ioutil.ReadDir(dir)
		if err != nil {
			t.Fatal("Unable to read directory", dir)
			return ""
		}

		found := false
		for _, f := range files {
			if strings.HasSuffix(f.Name(), "go.mod") {
				fp, _ = filepath.Abs(dir)

				// found the project root
				args = []string{
					"build", "-o", "./bin/shipyardtest",
					filepath.Join(fp, "main.go"),
				}

				found = true
				break
			}
		}

		if found {
			break
		}

		// check the parent
		dir = path.Join(dir, "../")
		parent++
	}

	if len(args) == 0 {
		t.Fatal("Unable to build test binary")
	}

	cmd := exec.Command("go", args...)
	cmd.Dir = fp

	err := cmd.Start()
	if err != nil {
		t.Fatal(err)
	}

	err = cmd.Wait()
	if err != nil {
		t.Fatal(err)
	}

	return filepath.Join(fp, "./bin/shipyardtest")
}

var suiteTemp string
var suiteBinary string
var suiteCertBundle *CertBundle
var suiteOptions ConnectorOptions

func TestConnectorSuite(t *testing.T) {
	suiteTemp = t.TempDir()
	suiteBinary = buildConnector(t)

	suiteOptions.LogDirectory = suiteTemp
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
		"tester",
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
	assert.FileExists(t, path.Join(suiteTemp, "connector.log"))

	// check is running
	assert.Eventually(t, func() bool {
		return c.IsRunning()
	}, 3*time.Second, 100*time.Millisecond)
}
