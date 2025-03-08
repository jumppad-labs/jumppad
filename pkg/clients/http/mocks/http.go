// Code generated by mockery v2.53.0. DO NOT EDIT.

package mocks

import (
	http "net/http"
	time "time"

	mock "github.com/stretchr/testify/mock"
)

// HTTP is an autogenerated mock type for the HTTP type
type HTTP struct {
	mock.Mock
}

// Do provides a mock function with given fields: r
func (_m *HTTP) Do(r *http.Request) (*http.Response, error) {
	ret := _m.Called(r)

	if len(ret) == 0 {
		panic("no return value specified for Do")
	}

	var r0 *http.Response
	var r1 error
	if rf, ok := ret.Get(0).(func(*http.Request) (*http.Response, error)); ok {
		return rf(r)
	}
	if rf, ok := ret.Get(0).(func(*http.Request) *http.Response); ok {
		r0 = rf(r)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*http.Response)
		}
	}

	if rf, ok := ret.Get(1).(func(*http.Request) error); ok {
		r1 = rf(r)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// HealthCheckHTTP provides a mock function with given fields: uri, method, headers, body, codes, timeout
func (_m *HTTP) HealthCheckHTTP(uri string, method string, headers map[string][]string, body string, codes []int, timeout time.Duration) error {
	ret := _m.Called(uri, method, headers, body, codes, timeout)

	if len(ret) == 0 {
		panic("no return value specified for HealthCheckHTTP")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, map[string][]string, string, []int, time.Duration) error); ok {
		r0 = rf(uri, method, headers, body, codes, timeout)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// HealthCheckTCP provides a mock function with given fields: uri, timeout
func (_m *HTTP) HealthCheckTCP(uri string, timeout time.Duration) error {
	ret := _m.Called(uri, timeout)

	if len(ret) == 0 {
		panic("no return value specified for HealthCheckTCP")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string, time.Duration) error); ok {
		r0 = rf(uri, timeout)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewHTTP creates a new instance of HTTP. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewHTTP(t interface {
	mock.TestingT
	Cleanup(func())
}) *HTTP {
	mock := &HTTP{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
