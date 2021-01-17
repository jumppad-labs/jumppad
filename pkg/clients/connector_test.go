package clients

import (
	"fmt"
	"io/ioutil"
	"math/rand"
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
			fmt.Println("dir", dir, f.Name())
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
var suiteOptions = ConnectorOptions{}

func TestConnectorSuite(t *testing.T) {
	suiteTemp = t.TempDir()
	suiteBinary = buildConnector(t)

	suiteOptions.LogDirectory = suiteTemp
	suiteOptions.BinaryPath = suiteBinary
	suiteOptions.RootCertPath = path.Join(suiteTemp, "root.cert")
	suiteOptions.LeafCertPath = path.Join(suiteTemp, "leaf.cert")
	suiteOptions.LeafKeyPath = path.Join(suiteTemp, "leaf.key")
	suiteOptions.GrpcBind = fmt.Sprintf(":%d", rand.Intn(1000)+20000)
	suiteOptions.HTTPBind = fmt.Sprintf(":%d", rand.Intn(1000)+20000)

	t.Run("Starts generates certificates", testGenerateCreatesBundle)
	t.Run("Starts Connector correctly", testStartsConnector)
}

func testGenerateCreatesBundle(t *testing.T) {
	c := NewConnector(suiteOptions)

	err := c.GenerateLocalBundle(suiteTemp)
	assert.NoError(t, err)

	assert.FileExists(t, path.Join(suiteTemp, "root.key"))
	assert.FileExists(t, suiteOptions.RootCertPath)
	assert.FileExists(t, suiteOptions.LeafKeyPath)
	assert.FileExists(t, suiteOptions.LeafCertPath)
}

func testStartsConnector(t *testing.T) {
	c := NewConnector(suiteOptions)

	err := c.Start()
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
