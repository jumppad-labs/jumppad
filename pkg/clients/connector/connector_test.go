package connector

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jumppad-labs/connector/protos/shipyard"
	"github.com/jumppad-labs/jumppad/pkg/clients/connector/mocks"
	"github.com/jumppad-labs/jumppad/pkg/clients/connector/types"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/stretchr/testify/mock"
	assert "github.com/stretchr/testify/require"
)

var suiteTemp string
var suiteCertBundle *types.CertBundle
var suiteOptions ConnectorOptions
var testBinaryPath string

func TestMain(m *testing.M) {
	// Build or reuse cached test helper binary
	var err error
	testBinaryPath, err = getOrBuildTestHelper()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get test helper: %v\n", err)
		os.Exit(1)
	}

	// Run all tests
	exitCode := m.Run()

	os.Exit(exitCode)
}

func getOrBuildTestHelper() (string, error) {
	// Get the directory where this test file is located
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("unable to get current test file path")
	}
	testDir := filepath.Dir(filename)
	testdataDir := filepath.Join(testDir, "testdata")

	// Source and binary paths
	helperSource := filepath.Join(testdataDir, "connector-test-helper.go")
	binaryName := "connector-test-helper"
	if runtime.GOOS == "windows" {
		binaryName = "connector-test-helper.exe"
	}
	binaryPath := filepath.Join(testdataDir, binaryName)

	// Check if binary exists and is up-to-date
	needsBuild := false
	binaryStat, err := os.Stat(binaryPath)
	if os.IsNotExist(err) {
		needsBuild = true
	} else if err != nil {
		return "", fmt.Errorf("failed to stat binary: %w", err)
	} else {
		// Check if source is newer than binary
		sourceStat, err := os.Stat(helperSource)
		if err != nil {
			return "", fmt.Errorf("failed to stat source: %w", err)
		}
		if sourceStat.ModTime().After(binaryStat.ModTime()) {
			needsBuild = true
		}
	}

	if needsBuild {
		// Ensure testdata directory exists
		if err := os.MkdirAll(testdataDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create testdata dir: %w", err)
		}

		// Build the test helper
		cmd := exec.Command("go", "build", "-o", binaryPath, helperSource)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("failed to build test helper: %w\nOutput: %s", err, output)
		}
	}

	return binaryPath, nil
}

func getBinaryPath(t *testing.T) string {
	if testBinaryPath == "" {
		t.Fatal("test binary was not initialized in TestMain")
	}
	return testBinaryPath
}

func TestConnectorSuite(t *testing.T) {
	suiteTemp = t.TempDir()

	tmpBinary := getBinaryPath(t)

	os.Setenv(utils.HomeEnvName(), suiteTemp)

	suiteOptions.LogDirectory = path.Join(suiteTemp, "logs")
	suiteOptions.BinaryPath = tmpBinary
	suiteOptions.GrpcBind = fmt.Sprintf(":%d", rand.Intn(1000)+20000)
	suiteOptions.HTTPBind = fmt.Sprintf(":%d", rand.Intn(1000)+20000)

	os.MkdirAll(suiteOptions.LogDirectory, os.ModePerm)

	t.Run("Generates certificates", testGenerateCreatesBundle)
	t.Run("Fetches certificates", testFetchesLocalCertBundle)
	t.Run("Generates a leaf certificate", testGenerateCreatesLeaf)
	t.Run("Starts Connector correctly", testStartsConnector)
	t.Run("Calls expose", testExposeServiceCallsExpose)
	t.Run("Calls remove", testRemoveServiceCallsRemove)
	t.Run("Calls list", testListServicesCallsList)
	t.Run("Starts with path containing spaces", testConnectorStartPathWithSpaces)
}

func testGenerateCreatesBundle(t *testing.T) {
	c := NewConnector(suiteOptions)

	var err error
	suiteCertBundle, err = c.GenerateLocalCertBundle(utils.CertsDir(""))
	assert.NoError(t, err)

	assert.FileExists(t, suiteCertBundle.RootCertPath)
	assert.FileExists(t, suiteCertBundle.RootKeyPath)
	assert.FileExists(t, suiteCertBundle.LeafKeyPath)
	assert.FileExists(t, suiteCertBundle.LeafCertPath)
}

func testFetchesLocalCertBundle(t *testing.T) {
	c := NewConnector(suiteOptions)

	cb, err := c.GetLocalCertBundle(utils.CertsDir(""))
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

	logFile := path.Join(suiteTemp, "logs", "connector.log")

	t.Cleanup(func() {
		c.Stop()

		if t.Failed() {
			d, _ := os.ReadFile(logFile)
			fmt.Println(string(d))
		}
	})

	// check the logfile
	assert.FileExists(t, logFile)

	// check is running
	assert.Eventually(t, func() bool {
		return c.IsRunning()
	}, 15*time.Second, 100*time.Millisecond)
}

func testExposeServiceCallsExpose(t *testing.T) {
	// ensure the socket has been released from the previous test
	assert.Eventually(t, func() bool {
		_, err := net.Dial("tcp", suiteOptions.GrpcBind)
		return err != nil
	}, 2*time.Second, 100*time.Millisecond)

	ts := mocks.NewMockConnectorServer()
	ts.On("ExposeService", mock.Anything, mock.Anything).Return(&shipyard.ExposeResponse{}, nil)

	_, err := ts.Start(suiteOptions.GrpcBind, suiteCertBundle.RootCertPath, suiteCertBundle.RootKeyPath, suiteCertBundle.LeafCertPath, suiteCertBundle.LeafKeyPath)
	assert.NoError(t, err)

	t.Cleanup(func() {
		ts.Stop()
	})

	r := &shipyard.ExposeRequest{}
	r.Service = &shipyard.Service{
		Name:                "test",
		RemoteConnectorAddr: "remoteaddr",
		DestinationAddr:     "destaddr",
		SourcePort:          8080,
		Type:                shipyard.ServiceType_REMOTE,
	}

	c := NewConnector(suiteOptions)
	_, err = c.ExposeService(
		r.Service.Name,
		int(r.Service.SourcePort),
		r.Service.RemoteConnectorAddr,
		r.Service.DestinationAddr,
		"remote",
	)
	assert.NoError(t, err)

	ts.AssertCalled(t, "ExposeService", mock.Anything, r)
}

func testRemoveServiceCallsRemove(t *testing.T) {
	// ensure the socket has been released from the previous test
	assert.Eventually(t, func() bool {
		_, err := net.Dial("tcp", suiteOptions.GrpcBind)
		return err != nil
	}, 2*time.Second, 100*time.Millisecond)

	ts := mocks.NewMockConnectorServer()
	ts.On("DestroyService", mock.Anything, mock.Anything).Return(&shipyard.NullMessage{}, nil)

	_, err := ts.Start(suiteOptions.GrpcBind, suiteCertBundle.RootCertPath, suiteCertBundle.RootKeyPath, suiteCertBundle.LeafCertPath, suiteCertBundle.LeafKeyPath)
	assert.NoError(t, err)

	t.Cleanup(func() {
		ts.Stop()
	})

	r := &shipyard.DestroyRequest{}
	r.Id = "tester"

	c := NewConnector(suiteOptions)
	err = c.RemoveService(r.Id)
	assert.NoError(t, err)

	ts.AssertCalled(t, "DestroyService", mock.Anything, r)
}

func testListServicesCallsList(t *testing.T) {
	// ensure the socket has been released from the previous test
	assert.Eventually(t, func() bool {
		_, err := net.Dial("tcp", suiteOptions.GrpcBind)
		return err != nil
	}, 2*time.Second, 100*time.Millisecond)

	ts := mocks.NewMockConnectorServer()
	ts.On("ListServices", mock.Anything, mock.Anything).Return(
		&shipyard.ListResponse{
			Services: []*shipyard.Service{&shipyard.Service{Id: "tester"}},
		},
		nil,
	)

	_, err := ts.Start(suiteOptions.GrpcBind, suiteCertBundle.RootCertPath, suiteCertBundle.RootKeyPath, suiteCertBundle.LeafCertPath, suiteCertBundle.LeafKeyPath)
	assert.NoError(t, err)

	t.Cleanup(func() {
		ts.Stop()
	})

	c := NewConnector(suiteOptions)
	svc, err := c.ListServices()
	assert.NoError(t, err)
	assert.Len(t, svc, 1)

	ts.AssertCalled(t, "ListServices", mock.Anything, mock.Anything)
}

func testConnectorStartPathWithSpaces(t *testing.T) {
	// This test verifies that binary paths containing spaces work correctly
	// This would have caught issue #358

	localTemp := t.TempDir()
	os.Setenv(utils.HomeEnvName(), localTemp)

	// Create a directory path with spaces
	dirWithSpaces := filepath.Join(localTemp, "path with spaces")
	err := os.MkdirAll(dirWithSpaces, 0755)
	assert.NoError(t, err)

	// Copy test binary to the space-containing path
	testBinary := getBinaryPath(t)
	binaryName := filepath.Base(testBinary)
	binaryInSpacePath := filepath.Join(dirWithSpaces, binaryName)

	// Copy the test binary
	input, err := os.ReadFile(testBinary)
	assert.NoError(t, err)
	err = os.WriteFile(binaryInSpacePath, input, 0755)
	assert.NoError(t, err)

	// Setup connector options with the space-containing path
	opts := ConnectorOptions{
		LogDirectory: filepath.Join(localTemp, "logs"),
		BinaryPath:   binaryInSpacePath,
		GrpcBind:     fmt.Sprintf(":%d", rand.Intn(1000)+20000),
		HTTPBind:     fmt.Sprintf(":%d", rand.Intn(1000)+20000),
		APIBind:      fmt.Sprintf(":%d", rand.Intn(1000)+20000),
		LogLevel:     "info",
		PidFile:      filepath.Join(localTemp, "connector.pid"),
	}
	os.MkdirAll(opts.LogDirectory, os.ModePerm)

	// Generate certificates
	c := NewConnector(opts)
	certBundle, err := c.GenerateLocalCertBundle(utils.CertsDir(""))
	assert.NoError(t, err)

	// Start connector with binary path containing spaces
	err = c.Start(certBundle)
	assert.NoError(t, err, "Connector should start successfully with spaces in binary path")

	// Cleanup
	t.Cleanup(func() {
		c.Stop()
	})

	// Verify connector is running
	assert.Eventually(t, func() bool {
		return c.IsRunning()
	}, 5*time.Second, 100*time.Millisecond, "Connector should be running after start")
}
