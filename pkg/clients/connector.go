package clients

import (
	"fmt"
	"os"
	"path"

	"github.com/shipyard-run/connector/crypto"
	"github.com/shipyard-run/gohup"
	"github.com/shipyard-run/shipyard/pkg/utils"
)

// Connector defines a client which can be used for interfacing with the
// Shipyard connector
type Connector interface {
	// Start the Connector, returns an error on failure
	Start(*CertBundle) error
	// Stop the Connector, returns an error on failure
	Stop() error
	// IsRunning returns true when the Connector is running
	IsRunning() bool

	// GenerateLocalCertBundle generates a root CA and leaf certificate for
	// securing connector communications for the local instance
	// this function is a convenience function which wraps other
	// methods
	GenerateLocalCertBundle(out string) (*CertBundle, error)

	// Fetches the local certificate bundle from the given directory
	// if any of the required files do not exist an error and a nil
	// CertBundle will be returned
	GetLocalCertBundle(dir string) (*CertBundle, error)

	// Generates a Leaf certificate for securing a connector
	GenerateLeafCert(privateKey, rootCA, hosts string, ips []string, dir string) (*CertBundle, error)
}

var defaultArgs = []string{
	"connector",
	"--help",
}

// ConnectorImpl is a concrete implementation of the Connector interface
type ConnectorImpl struct {
	options ConnectorOptions
}

type ConnectorOptions struct {
	LogDirectory string
	BinaryPath   string
	GrpcBind     string
	HTTPBind     string
	LogLevel     string
	PidFile      string
}

type CertBundle struct {
	RootCertPath string
	RootKeyPath  string
	LeafCertPath string
	LeafKeyPath  string
}

func DefaultConnectorOptions() ConnectorOptions {
	co := ConnectorOptions{}
	co.LogDirectory = utils.LogsDir()
	co.BinaryPath = utils.GetShipyardBinaryPath()
	co.GrpcBind = ":30001"
	co.HTTPBind = ":30002"
	co.LogLevel = "info"
	co.PidFile = utils.GetConnectorPIDFile()

	return co
}

// NewConnector creates a new connector with the given options
func NewConnector(opts ConnectorOptions) Connector {
	return &ConnectorImpl{options: opts}
}

// Start the Connector, returns an error on failure
func (c *ConnectorImpl) Start(cb *CertBundle) error {
	lp := &gohup.LocalProcess{}
	o := gohup.Options{
		Path: c.options.BinaryPath,
		Args: []string{
			"connector",
			"run",
			"--grpc-bind", c.options.GrpcBind,
			"--http-bind", c.options.HTTPBind,
			"--root-cert-path", cb.RootCertPath,
			"--server-cert-path", cb.LeafCertPath,
			"--server-key-path", cb.LeafKeyPath,
		},
		Logfile: path.Join(c.options.LogDirectory, "connector.log"),
		Pidfile: c.options.PidFile,
	}

	var err error
	_, c.options.PidFile, err = lp.Start(o)
	return err
}

// Stop the Connector, returns an error on failure
func (c *ConnectorImpl) Stop() error {
	lp := &gohup.LocalProcess{}
	return lp.Stop(c.options.PidFile)
}

// IsRunning returns true when the Connector is running
func (c *ConnectorImpl) IsRunning() bool {
	lp := &gohup.LocalProcess{}
	status, err := lp.QueryStatus(c.options.PidFile)
	if err != nil {
		return false
	}

	if status == gohup.StatusRunning {
		return true
	}

	return false
}

// creates a CA and local leaf cert
func (c *ConnectorImpl) GenerateLocalCertBundle(out string) (*CertBundle, error) {
	cb := &CertBundle{
		RootCertPath: path.Join(out, "root.cert"),
		RootKeyPath:  path.Join(out, "root.key"),
		LeafCertPath: path.Join(out, "leaf.cert"),
		LeafKeyPath:  path.Join(out, "leaf.key"),
	}

	fmt.Println(out)

	// create the CA
	rk, err := crypto.GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	ca, err := crypto.GenerateCA(rk.Private)
	if err != nil {
		return nil, err
	}

	os.Remove(cb.RootKeyPath)
	err = rk.Private.WriteFile(cb.RootKeyPath)
	if err != nil {
		return nil, err
	}

	os.Remove(cb.RootCertPath)
	err = ca.WriteFile(cb.RootCertPath)
	if err != nil {
		return nil, err
	}

	ips := utils.GetLocalIPAddresses()
	host := utils.GetHostname()

	return c.GenerateLeafCert(cb.RootKeyPath, cb.RootCertPath, host, ips, out)
}

func (c *ConnectorImpl) GetLocalCertBundle(dir string) (*CertBundle, error) {
	cb := &CertBundle{
		RootCertPath: path.Join(dir, "root.cert"),
		RootKeyPath:  path.Join(dir, "root.key"),
		LeafCertPath: path.Join(dir, "leaf.cert"),
		LeafKeyPath:  path.Join(dir, "leaf.key"),
	}

	// test to see if files exist
	f1, err := os.Open(cb.RootCertPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("Unable to find root certificate")
	}
	defer f1.Close()

	f2, err := os.Open(cb.RootKeyPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("Unable to find root key")
	}
	defer f2.Close()

	f3, err := os.Open(cb.LeafCertPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("Unable to find leaf certificate")
	}
	defer f3.Close()

	f4, err := os.Open(cb.LeafKeyPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("Unable to find leaf key")
	}
	defer f4.Close()

	return cb, nil
}

// GenerateLeafCert generates a x509 leaf certificate with the given details
func (c *ConnectorImpl) GenerateLeafCert(
	rootKey, rootCA, host string, ips []string, dir string) (*CertBundle, error) {

	cb := &CertBundle{
		RootCertPath: rootCA,
		RootKeyPath:  rootKey,
		LeafCertPath: path.Join(dir, "leaf.cert"),
		LeafKeyPath:  path.Join(dir, "leaf.key"),
	}

	// load the root key
	rk := &crypto.PrivateKey{}
	err := rk.ReadFile(cb.RootKeyPath)
	if err != nil {
		return nil, err
	}

	// load the ca
	ca := &crypto.X509{}
	err = ca.ReadFile(cb.RootCertPath)
	if err != nil {
		return nil, err
	}

	// generate a local cert
	k, err := crypto.GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	os.Remove(cb.LeafKeyPath)
	err = k.Private.WriteFile(cb.LeafKeyPath)
	if err != nil {
		return nil, err
	}

	lc, err := crypto.GenerateLeaf(
		ips,
		[]string{"localhost", "*.shipyard.run", host},
		ca,
		rk,
		k.Private)
	if err != nil {
		return nil, err
	}

	os.Remove(cb.LeafCertPath)
	err = lc.WriteFile(cb.LeafCertPath)
	if err != nil {
		return nil, err
	}

	return cb, nil
}
