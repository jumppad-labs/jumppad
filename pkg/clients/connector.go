package clients

import (
	"fmt"
	"path"

	"github.com/shipyard-run/connector/crypto"
	"github.com/shipyard-run/gohup"
	"github.com/shipyard-run/shipyard/pkg/utils"
)

// Connector defines a client which can be used for interfacing with the
// Shipyard connector
type Connector interface {
	// Start the Connector, returns an error on failure
	Start() error
	// Stop the Connector, returns an error on failure
	Stop() error
	// IsRunning returns true when the Connector is running
	IsRunning() bool

	// GenerateLocalBundle generates a root CA and leaf certificate for
	// securing connector communications for the local instance
	// this function is a convenience function which wraps other
	// methods
	GenerateLocalBundle(out string) error
}

var defaultArgs = []string{
	"connector",
	"--help",
}

// ConnectorImpl is a concrete implementation of the Connector interface
type ConnectorImpl struct {
	pid     int
	pidfile string
	options ConnectorOptions
}

type ConnectorOptions struct {
	LogDirectory string
	BinaryPath   string
	GrpcBind     string
	HTTPBind     string
	LogLevel     string
	RootCertPath string
	LeafCertPath string
	LeafKeyPath  string
}

// NewConnector creates a new connector with the given options
func NewConnector(opts ConnectorOptions) Connector {
	return &ConnectorImpl{options: opts}
}

// Start the Connector, returns an error on failure
func (c *ConnectorImpl) Start() error {
	lp := &gohup.LocalProcess{}
	o := gohup.Options{
		Path: c.options.BinaryPath,
		Args: []string{
			"connector",
			"run",
			"--grpc-bind", c.options.GrpcBind,
			"--http-bind", c.options.HTTPBind,
			"--root-cert-path", c.options.RootCertPath,
			"--server-cert-path", c.options.RootCertPath,
			"--server-key-path", c.options.RootCertPath,
		},
		Logfile: path.Join(c.options.LogDirectory, "connector.log"),
	}

	var err error
	c.pid, c.pidfile, err = lp.Start(o)
	if err != nil {
		panic(err)
	}

	return err
}

// Stop the Connector, returns an error on failure
func (c *ConnectorImpl) Stop() error {
	lp := &gohup.LocalProcess{}
	return lp.Stop(c.pidfile)
}

// IsRunning returns true when the Connector is running
func (c *ConnectorImpl) IsRunning() bool {
	lp := &gohup.LocalProcess{}
	status, err := lp.QueryStatus(c.pidfile)
	if err != nil {
		return false
	}

	if status == gohup.StatusRunning {
		return true
	}

	return false
}

// creates a CA and local leaf cert
func (c *ConnectorImpl) GenerateLocalBundle(out string) error {
	// create the CA
	rk, err := crypto.GenerateKeyPair()
	if err != nil {
		return err
	}

	ca, err := crypto.GenerateCA(rk.Private)
	if err != nil {
		return err
	}

	err = rk.Private.WriteFile(path.Join(out, "root.key"))
	if err != nil {
		return err
	}

	err = ca.WriteFile(path.Join(out, "root.cert"))
	if err != nil {
		return err
	}

	// generate a local cert
	k, err := crypto.GenerateKeyPair()
	if err != nil {
		return err
	}

	err = k.Private.WriteFile(path.Join(out, "leaf.key"))
	if err != nil {
		return err
	}

	ips := utils.GetLocalIPAddresses()
	host := utils.GetHostname()

	fmt.Println(ips, host)

	lc, err := crypto.GenerateLeaf(
		ips,
		[]string{"localhost", "*.shipyard.run", host},
		ca,
		rk.Private,
		k.Private)
	if err != nil {
		return err
	}

	err = lc.WriteFile(path.Join(out, "leaf.cert"))
	return nil
}
