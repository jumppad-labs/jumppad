package connector

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"path"
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
var suiteBinary string
var suiteCertBundle *types.CertBundle
var suiteOptions ConnectorOptions

func TestConnectorSuite(t *testing.T) {
	suiteTemp = t.TempDir()
	suiteBinary = utils.GetJumppadBinaryPath()

	home := os.Getenv(utils.HomeEnvName())
	os.Setenv(utils.HomeEnvName(), suiteTemp)
	t.Cleanup(func() {
		os.Setenv(utils.HomeEnvName(), home)
	})

	suiteOptions.LogDirectory = path.Join(suiteTemp, "logs")
	suiteOptions.BinaryPath = suiteBinary
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
