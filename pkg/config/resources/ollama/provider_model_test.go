package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/jumppad-labs/hclconfig/types"
	httpmocks "github.com/jumppad-labs/jumppad/pkg/clients/http/mocks"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func createProvider(t *testing.T, model string, httpClient *httpmocks.HTTP) *ModelProvider {
	config := &OllamaModel{
		ResourceBase: types.ResourceBase{
			Meta: types.Meta{
				Name: "test",
			},
		},
		Model: model,
	}

	return &ModelProvider{
		config:     config,
		log:        logger.NewTestLogger(t),
		httpClient: httpClient,
	}
}

func createJSONResponse(statusCode int, data any) *http.Response {
	jsonData, _ := json.Marshal(data)
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewReader(jsonData)),
		Header:     make(http.Header),
	}
}

func createStringResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestModelProviderCreateWhenModelExists(t *testing.T) {
	httpMock := &httpmocks.HTTP{}
	provider := createProvider(t, "llama2:7b", httpMock)

	// Mock response for /api/tags showing model exists
	tagsResponse := map[string]any{
		"models": []map[string]any{
			{
				"name":   "llama2:7b",
				"digest": "sha256:test-digest",
				"size":   int64(1024),
			},
		},
	}

	httpMock.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.Method == "GET" && strings.Contains(req.URL.Path, "/api/tags")
	})).Return(createJSONResponse(200, tagsResponse), nil)

	err := provider.Create(context.Background())
	require.NoError(t, err)

	// Verify the mock was called
	httpMock.AssertExpectations(t)
}

func TestModelProviderCreatePullsModel(t *testing.T) {
	httpMock := &httpmocks.HTTP{}
	provider := createProvider(t, "llama2:7b", httpMock)

	// First call to /api/tags shows no model
	emptyTagsResponse := map[string]any{"models": []map[string]any{}}

	// Mock the first tags call (no models)
	httpMock.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.Method == "GET" && strings.Contains(req.URL.Path, "/api/tags")
	})).Return(createJSONResponse(200, emptyTagsResponse), nil).Once()

	// Mock pull response with streaming JSON
	pullResponse := `{"status":"pulling manifest"}
{"status":"downloading","completed":50.0,"total":100.0}
{"status":"verifying","digest":"sha256:new-digest"}`

	httpMock.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.Method == "POST" && strings.Contains(req.URL.Path, "/api/pull")
	})).Return(createStringResponse(200, pullResponse), nil).Once()

	// Mock second tags call after pull (with model)
	finalTagsResponse := map[string]any{
		"models": []map[string]any{
			{
				"name":   "llama2:7b",
				"digest": "sha256:new-digest",
				"size":   int64(2048),
			},
		},
	}

	httpMock.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.Method == "GET" && strings.Contains(req.URL.Path, "/api/tags")
	})).Return(createJSONResponse(200, finalTagsResponse), nil).Once()

	err := provider.Create(context.Background())
	require.NoError(t, err)
	require.Equal(t, "sha256:new-digest", provider.config.Digest)
	require.Equal(t, int64(2048), provider.config.Size)

	// Verify all mocks were called
	httpMock.AssertExpectations(t)
}

func TestModelProviderCreatePullError(t *testing.T) {
	httpMock := &httpmocks.HTTP{}
	provider := createProvider(t, "llama2:7b", httpMock)

	// Mock empty tags response
	emptyTagsResponse := map[string]any{"models": []map[string]any{}}
	httpMock.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.Method == "GET" && strings.Contains(req.URL.Path, "/api/tags")
	})).Return(createJSONResponse(200, emptyTagsResponse), nil).Once()

	// Mock pull error response
	httpMock.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.Method == "POST" && strings.Contains(req.URL.Path, "/api/pull")
	})).Return(createStringResponse(500, "Internal Server Error"), nil).Once()

	err := provider.Create(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to pull model")

	httpMock.AssertExpectations(t)
}

func TestModelProviderRefresh(t *testing.T) {
	httpMock := &httpmocks.HTTP{}
	provider := createProvider(t, "llama2:7b", httpMock)
	provider.config.Digest = "sha256:old-digest"
	provider.config.Size = 1024

	// Mock response for /api/tags showing updated model
	tagsResponse := map[string]any{
		"models": []map[string]any{
			{
				"name":   "llama2:7b",
				"digest": "sha256:refreshed-digest",
				"size":   int64(4096),
			},
		},
	}

	httpMock.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.Method == "GET" && strings.Contains(req.URL.Path, "/api/tags")
	})).Return(createJSONResponse(200, tagsResponse), nil)

	err := provider.Refresh(context.Background())
	require.NoError(t, err)
	require.Equal(t, "sha256:refreshed-digest", provider.config.Digest)
	require.Equal(t, int64(4096), provider.config.Size)

	httpMock.AssertExpectations(t)
}

func TestModelProviderRefreshModelNotFound(t *testing.T) {
	httpMock := &httpmocks.HTTP{}
	provider := createProvider(t, "llama2:7b", httpMock)

	// Mock response for /api/tags showing no models
	emptyTagsResponse := map[string]any{"models": []map[string]any{}}

	httpMock.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.Method == "GET" && strings.Contains(req.URL.Path, "/api/tags")
	})).Return(createJSONResponse(200, emptyTagsResponse), nil)

	err := provider.Refresh(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "model llama2:7b no longer exists")

	httpMock.AssertExpectations(t)
}
