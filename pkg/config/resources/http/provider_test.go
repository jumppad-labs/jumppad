package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/instruqt/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/stretchr/testify/require"
)

func setupHttp(t *testing.T) (*HTTP, *Provider) {
	h := &HTTP{ResourceBase: types.ResourceBase{Meta: types.Meta{Name: "test"}}}

	p := &Provider{h, logger.NewTestLogger(t), *http.DefaultClient}

	return h, p
}

func TestHttpResourceGet(t *testing.T) {
	expected := "your response"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, expected)
	}))

	defer ts.Close()

	h, p := setupHttp(t)
	h.Method = "GET"
	h.URL = ts.URL

	err := p.Create(context.Background())
	require.NoError(t, err)

	require.Equal(t, 200, h.Status)
	require.Equal(t, expected, h.Body)
}

func TestHttpResourcePost(t *testing.T) {
	var body string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		body = string(data)
	}))

	defer ts.Close()

	h, p := setupHttp(t)
	h.Method = "POST"
	h.URL = ts.URL
	h.Payload = "your request"

	err := p.Create(context.Background())
	require.NoError(t, err)

	require.Equal(t, 200, h.Status)
	require.Equal(t, h.Payload, body)
}

func TestHttpResourceHeaders(t *testing.T) {
	headers := map[string]string{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers["X-First"] = r.Header.Get("X-First")
		headers["X-Second"] = r.Header.Get("X-Second")
	}))

	defer ts.Close()

	h, p := setupHttp(t)
	h.Method = "POST"
	h.URL = ts.URL
	h.Headers = map[string]string{
		"X-First":  "first",
		"X-Second": "second",
	}

	err := p.Create(context.Background())
	require.NoError(t, err)

	require.Equal(t, 200, h.Status)
	require.Equal(t, h.Headers, headers)
}

func TestHttpResourceTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// do nothing
		time.Sleep(10 * time.Second)
	}))

	defer ts.Close()

	h, p := setupHttp(t)
	h.Method = "GET"
	h.URL = ts.URL
	h.Timeout = "5s"

	err := p.Create(context.Background())
	require.Error(t, err)
}
