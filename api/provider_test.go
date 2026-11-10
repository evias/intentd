package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewProvider tests [Provider] construction and accessors.
func TestNewProvider(t *testing.T) {
	p := NewProvider("OpenAI", "https://api.openai.com/v1")

	assert.Equal(t, "OpenAI", p.Name())
	assert.Equal(t, APITypeOpenAI, p.Type())
	assert.Equal(t, "https://api.openai.com/v1", p.BaseURL())
}

// TestProvider_BaseURL tests trailing-slash normalization of the base url.
func TestProvider_BaseURL(t *testing.T) {
	testCases := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{name: "without_trailing_slash", baseURL: "https://api.openai.com/v1", expected: "https://api.openai.com/v1"},
		{name: "with_trailing_slash", baseURL: "https://api.openai.com/v1/", expected: "https://api.openai.com/v1"},
		{name: "empty", baseURL: "", expected: ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewProvider("Test", tc.baseURL)
			assert.Equal(t, tc.expected, p.BaseURL())
		})
	}
}

// TestProvider_WithOptions tests the [ProviderOption] helpers.
func TestProvider_WithOptions(t *testing.T) {
	p := NewProvider("Test", "https://api.openai.com/v1",
		WithAPIType(APITypeOpenAI),
		WithAPIKey("secret-key"),
	)

	// apiType is exposed through the interface.
	assert.Equal(t, APITypeOpenAI, p.Type())

	// apiKey is only accessible through the private implementation.
	lp, ok := p.(*llmProvider)
	require.True(t, ok)
	assert.Equal(t, "secret-key", lp.apiKey)
}

// TestProvider_GetClient_Panic tests that getClient panics for apiTypes
// which are not yet implemented.
func TestProvider_GetClient_Panic(t *testing.T) {
	p := NewProvider("Test", "https://api.openai.com/v1",
		WithAPIType(APIType("not-yet-implemented")),
	)

	assert.Panics(t, func() {
		p.Call("gpt-4", Request{})
	})
}

// newTestServer creates a fake OpenAI-compatible http server returning the
// given response body, optionally capturing the received request body.
func newTestServer(t *testing.T, response string, received *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if received != nil {
			body, _ := io.ReadAll(r.Body)
			*received = string(body)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
}

// TestProvider_Call tests [Provider.Call] against a fake OpenAI-compatible
// http server, using the provider's forwarded baseURL.
func TestProvider_Call(t *testing.T) {
	t.Run("text_message", func(t *testing.T) {
		var receivedBody string
		srv := newTestServer(t, `{"choices":[{"index":0,"message":{"role":"assistant","content":"Hello!"}}]}`, &receivedBody)
		defer srv.Close()

		p := NewProvider("Test", srv.URL, WithAPIKey("test-key"))
		resp, err := p.Call("gpt-4", Request{
			NewMessage("text/plain", []byte("Hi there")),
		})
		require.NoError(t, err)
		require.Len(t, resp, 1)

		assert.True(t, resp[0].IsText())
		assert.Equal(t, "Hello!", resp[0].String())

		// The request body must contain the sent user message.
		assert.Contains(t, receivedBody, "Hi there")
		assert.Contains(t, receivedBody, "gpt-4")
	})

	t.Run("non_text_messages_are_filtered", func(t *testing.T) {
		var receivedBody string
		srv := newTestServer(t, `{"choices":[{"index":0,"message":{"role":"assistant","content":""}}]}`, &receivedBody)
		defer srv.Close()

		p := NewProvider("Test", srv.URL, WithAPIKey("test-key"))
		resp, err := p.Call("gpt-4", Request{
			NewMessage("image/png", []byte("fake-image-bytes")),
		})
		require.NoError(t, err)
		require.Len(t, resp, 1)

		// Only text messages are appended to the chat completion request.
		assert.False(t, strings.Contains(receivedBody, "fake-image-bytes"))
	})

	t.Run("http_error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
		}))
		defer srv.Close()

		p := NewProvider("Test", srv.URL, WithAPIKey("invalid-key"))
		resp, err := p.Call("gpt-4", Request{
			NewMessage("text/plain", []byte("Hi there")),
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "openai: failed to execute chat completion")
		assert.Empty(t, resp)
	})
}
