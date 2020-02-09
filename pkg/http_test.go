package clients

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPGetReturnsParams(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(rwhttp.ResponseWriter, *http.Request) {

	}))
}
