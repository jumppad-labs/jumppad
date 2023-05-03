package mocks

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net"

	"github.com/jumppad-labs/connector/protos/shipyard"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type MockConnectorServer struct {
	mock.Mock
	server   *grpc.Server
	listener net.Listener
}

func NewMockConnectorServer() *MockConnectorServer {
	return &MockConnectorServer{}
}

// Start the server returning the location
func (m *MockConnectorServer) Start(addr, rootCertPath, rootKeyPath, leafCertPath, leafKeyPath string) (string, error) {
	certificate, err := tls.LoadX509KeyPair(leafCertPath, leafKeyPath)
	if err != nil {
		return "", fmt.Errorf("could not load server key pair: %s", err)
	}

	// Create a certificate pool from the certificate authority
	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(rootCertPath)
	if err != nil {
		return "", fmt.Errorf("could not read ca certificate: %s", err)
	}

	// Append the client certificates from the CA
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		return "", errors.New("failed to append client certs")
	}

	creds := credentials.NewTLS(&tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{certificate},
		ClientCAs:    certPool,
	})

	m.server = grpc.NewServer(grpc.Creds(creds))

	m.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return "", fmt.Errorf("Unable to listen on address: %s error: %s", addr, err)
	}

	shipyard.RegisterRemoteConnectionServer(m.server, m)

	// start the gRPC server
	go m.server.Serve(m.listener)

	return addr, nil
}

func (m *MockConnectorServer) Stop() {
	m.server.Stop()
	m.listener.Close()
}

// implements the connector grpc interface

func (m *MockConnectorServer) OpenStream(svr shipyard.RemoteConnection_OpenStreamServer) error {
	args := m.Called(svr)
	return args.Error(0)
}

func (m *MockConnectorServer) ExposeService(ctx context.Context, r *shipyard.ExposeRequest) (*shipyard.ExposeResponse, error) {
	args := m.Called(ctx, r)

	if er, ok := args.Get(0).(*shipyard.ExposeResponse); ok {
		return er, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockConnectorServer) DestroyService(ctx context.Context, r *shipyard.DestroyRequest) (*shipyard.NullMessage, error) {
	args := m.Called(ctx, r)

	if er, ok := args.Get(0).(*shipyard.NullMessage); ok {
		return er, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockConnectorServer) ListServices(ctx context.Context, msg *shipyard.NullMessage) (*shipyard.ListResponse, error) {
	args := m.Called(ctx, msg)

	if er, ok := args.Get(0).(*shipyard.ListResponse); ok {
		return er, args.Error(1)
	}

	return nil, args.Error(1)
}
