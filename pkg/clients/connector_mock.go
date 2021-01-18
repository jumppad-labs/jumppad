package clients

import "github.com/stretchr/testify/mock"

type ConnectorMock struct {
	mock.Mock
}

// Start the Connector, returns an error on failure
func (m *ConnectorMock) Start(cb *CertBundle) error {
	args := m.Called(cb)

	return args.Error(0)
}

// Stop the Connector, returns an error on failure
func (m *ConnectorMock) Stop() error {
	args := m.Called()

	return args.Error(0)
}

// IsRunning returns true when the Connector is running
func (m *ConnectorMock) IsRunning() bool {
	args := m.Called()

	return args.Bool(0)
}

// GenerateLocalBundle generates a root CA and leaf certificate for
// securing connector communications for the local instance
// this function is a convenience function which wraps other
// methods
func (m *ConnectorMock) GenerateLocalCertBundle(out string) (*CertBundle, error) {
	args := m.Called(out)

	if cb, ok := args.Get(0).(*CertBundle); ok {
		return cb, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *ConnectorMock) GetLocalCertBundle(out string) (*CertBundle, error) {
	args := m.Called(out)

	if cb, ok := args.Get(0).(*CertBundle); ok {
		return cb, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *ConnectorMock) GenerateLeafCert(privateKey, rootCA, hosts string, ips []string, dir string) (*CertBundle, error) {
	args := m.Called(privateKey, rootCA, hosts, ips, dir)

	if cb, ok := args.Get(0).(*CertBundle); ok {
		return cb, args.Error(1)
	}

	return nil, args.Error(1)
}
