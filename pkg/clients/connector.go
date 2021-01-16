package clients

import (
	"github.com/shipyard-run/gohup"
	"github.com/shipyard-run/shipyard/pkg/utils"
)

var runPath = utils.GetShipyardBinaryPath()

var runArgs = []string{
	"connector",
	"run",
}

// Connector defines a client which can be used for interfacing with the
// Shipyard connector
type Connector interface {
	// Start the Connector, returns an error on failure
	Start() error
	// Stop the Connector, returns an error on failure
	Stop() error
	// IsRunning returns true when the Connector is running
	IsRunning() bool
}

// ConnectorImpl is a concrete implementation of the Connector interface
type ConnectorImpl struct {
	pid     int
	pidfile string
}

// NewConnector creates a new connector
func NewConnector() Connector {
	return &ConnectorImpl{}
}

// Start the Connector, returns an error on failure
func (c *ConnectorImpl) Start() error {
	lp := &gohup.LocalProcess{}
	o := gohup.Options{
		Path: "/usr/bin/tail",
		Args: []string{
			"-f",
			"/dev/null",
		},
		Logfile: "./process.log",
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
	return nil
}

// IsRunning returns true when the Connector is running
func (c *ConnectorImpl) IsRunning() bool {
	return false
}
