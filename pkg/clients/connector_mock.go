package clients

import "github.com/stretchr/testify/mock"

type ConnectorMock struct {
	mock.Mock
}

// Start the Connector, returns an error on failure
func (m *ConnectorMock) Start() error {
	args := m.Called()

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

	return args.Get(0).(*CertBundle), args.Error(1)
}

func (m *ConnectorMock) FetchLocalCertBundle(out string) (*CertBundle, error) {
	args := m.Called(out)

	return args.Get(0).(*CertBundle), args.Error(1)
}
