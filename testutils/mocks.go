package testutils

import "github.com/stretchr/testify/mock"

// RemoveOn is a utility function for removing Expectations from mock objects
func RemoveOn(m *mock.Mock, method string) {
	ec := m.ExpectedCalls
	rc := make([]*mock.Call, 0)

	for _, c := range ec {
		if c.Method != method {
			rc = append(rc, c)
		}
	}

	m.ExpectedCalls = rc
}

func GetCalls(m *mock.Mock, method string) []mock.Call {
	rc := make([]mock.Call, 0)
	for _, c := range m.Calls {
		if c.Method == method {
			rc = append(rc, c)
		}
	}

	return rc
}
