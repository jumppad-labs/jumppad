package connector

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/instruqt/jumppad/pkg/clients/connector/mocks"
	"github.com/instruqt/jumppad/pkg/clients/connector/types"
	"github.com/instruqt/jumppad/pkg/utils"
	"github.com/jumppad-labs/connector/protos/shipyard"
	"github.com/stretchr/testify/mock"
	assert "github.com/stretchr/testify/require"
)

var suiteTemp string
var suiteCertBundle *types.CertBundle
var suiteOptions ConnectorOptions

func getBinaryPath(t *testing.T) string {
	currentLevel := 0
	maxLevels := 10

	// we are running from a test so use go run main.go as the command
	dir, _ := os.Getwd()
	// walk backwards until we find the go.mod
	for {
		files, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}

		for _, f := range files {
			if strings.HasSuffix(f.Name(), "go.mod") {
				fp, _ := filepath.Abs(dir)
				// found the project root
				file := filepath.Join(fp, "main.go")
				return fmt.Sprintf("go run %s", file)
			}
		}

		// check the parent
		dir = filepath.Join(dir, "../")
		fmt.Println(dir)
		currentLevel++
		if currentLevel > maxLevels {
			t.Fatal("unable to find go.mod")
		}
	}
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
