package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetInternalTools tests the internal tool definitions exposed to the
// chat completion API.
func TestGetInternalTools(t *testing.T) {
	tools := GetInternalTools()
	require.Len(t, tools, 1)

	bz, err := json.Marshal(tools[0])
	require.NoError(t, err)

	encoded := string(bz)
	assert.Contains(t, encoded, `"name":"native_fetch"`)
	assert.Contains(t, encoded, `"url"`)
	assert.Contains(t, encoded, `"required":["url"]`)
}

// TestNativeFetch tests the internal native_fetch tool implementation.
func TestNativeFetch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("FETCHED CONTENT"))
		}))
		defer srv.Close()

		result, err := native_fetch(context.Background(), srv.URL)
		require.NoError(t, err)
		assert.Equal(t, "FETCHED CONTENT", result)
	})

	t.Run("connection_refused", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := srv.URL
		srv.Close() // ensure connection refused

		result, err := native_fetch(context.Background(), url)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "http get")
		assert.Empty(t, result)
	})

	t.Run("known_tools_fn_registry", func(t *testing.T) {
		// The native_fetch tool must be registered for the LLM tool call flow.
		fn, ok := internalToolsByName["native_fetch"]
		require.True(t, ok)
		require.NotNil(t, fn)

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("REGISTRY FETCH"))
		}))
		defer srv.Close()

		result, err := fn(context.Background(), srv.URL)
		require.NoError(t, err)
		assert.Equal(t, "REGISTRY FETCH", result)
	})
}
