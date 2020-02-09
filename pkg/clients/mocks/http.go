package mocks

import (
	"net/http"

	"github.com/stretchr/testify/mock"
)

// MockHTTP is a mock implementation of the HTTP client
// interface
type MockHTTP struct {
	mock.Mock
}

func (m *MockHTTP) Get(url string) (*http.Response, error) {
	args := m.Called(url)

	if r, ok := args.Get(0).(*http.Response); ok {
		return r, args.Error(1)
	}

	return nil, args.Error(1)
}
