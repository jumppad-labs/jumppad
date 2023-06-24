package mocks

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockHTTP is a mock implementation of the HTTP client
// interface
type MockHTTP struct {
	mock.Mock
}

func (m *MockHTTP) HealthCheckHTTP(uri string, codes []int, timeout time.Duration) error {
	args := m.Called(uri, codes, timeout)

	return args.Error(0)
}

func (m *MockHTTP) HealthCheckTCP(uri string, timeout time.Duration) error {
	args := m.Called(uri, timeout)

	return args.Error(0)
}

func (m *MockHTTP) Do(r *http.Request) (*http.Response, error) {
	args := m.Called(r)

	if rr, ok := args.Get(0).(*http.Response); ok {
		return rr, args.Error(1)
	}

	return &http.Response{
		StatusCode: http.StatusTeapot,
		Body:       ioutil.NopCloser(bytes.NewReader([]byte(""))),
	}, args.Error(1)
}
