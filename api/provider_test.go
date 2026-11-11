package api

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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
		p.Generate("chat", "gpt-4", Request{})
	})
}

// TestProvider_ClientCaching tests that getClient only creates the underlying
// client once, and reuses it on subsequent Generate calls.
func TestProvider_ClientCaching(t *testing.T) {
	p := NewProvider("Test", "https://api.openai.com/v1")

	lp, ok := p.(*llmProvider)
	require.True(t, ok)
	assert.Nil(t, lp.client) // no client before the first Generate call

	srv := newSSEServer(t, []string{
		sseBody("first"),  // 1st pass
		sseBody("second"), // 2nd pass
	}, nil)
	defer srv.Close()

	p2 := NewProvider("Test", srv.URL, WithAPIKey("test-key"))
	lp2, ok := p2.(*llmProvider)
	require.True(t, ok)

	_, err := p2.Generate(OutputTypeChat, "gpt-4", Request{
		NewMessage("text/plain", []byte("Hi there")),
	})
	require.NoError(t, err)
	require.NotNil(t, lp2.client)

	first := lp2.client
	_, err = p2.Generate(OutputTypeChat, "gpt-4", Request{
		NewMessage("text/plain", []byte("Hi there")),
	})
	require.NoError(t, err)

	// The same client instance must be reused.
	assert.Same(t, first, lp2.client)
}

// TestProvider_Generate_UnknownOutputType tests that an unknown [OutputType]
// falls back to the Chat generation flow.
func TestProvider_Generate_UnknownOutputType(t *testing.T) {
	var receivedBodies []string
	srv := newSSEServer(t, []string{
		sseBody("Hello!"),  // 1st pass
		sseBody("Replied"), // 2nd pass
	}, &receivedBodies)
	defer srv.Close()

	p := NewProvider("Test", srv.URL, WithAPIKey("test-key"))
	resp, err := p.Generate(OutputType("unknown"), "gpt-4", Request{
		NewMessage("text/plain", []byte("Hi there")),
	})
	require.NoError(t, err)
	require.Len(t, resp, 1)

	// Unknown output types are handled by the Chat flow.
	assert.True(t, resp[0].IsText())
	assert.Equal(t, "Hello!Replied", resp[0].String())
}

// sseBody returns a Server-Sent Events formatted chat completion stream body
// with the given delta contents, terminated by the [DONE] marker.
func sseBody(contents ...string) string {
	var sb strings.Builder
	for _, c := range contents {
		chunk := fmt.Sprintf(
			`{"id":"chatcmpl-test","object":"chat.completion.chunk","model":"gpt-4","choices":[{"index":0,"delta":{"content":%q}}]}`,
			c)
		sb.WriteString("data: " + chunk + "\n\n")
	}
	sb.WriteString("data: [DONE]\n\n")
	return sb.String()
}

// newSSEServer creates a fake OpenAI-compatible http server responding to
// each request with the next Server-Sent Events body from responses, and
// optionally capturing each received request body.
func newSSEServer(t *testing.T, responses []string, received *[]string) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	requestCount := 0

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		i := requestCount
		requestCount++
		mu.Unlock()

		if received != nil {
			body, _ := io.ReadAll(r.Body)
			*received = append(*received, string(body))
		}

		response := ""
		if i < len(responses) {
			response = responses[i]
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(response))
	}))
}

// TestProvider_Call tests [Provider.Call] against a fake OpenAI-compatible
// http server, using the provider's forwarded baseURL.
//
// The current implementation of [openAIClient.Send] performs 2 rounds of
// streaming Chat Completions API calls: a first pass with tool definitions
// (injecting potential `native_fetch` calls), and a second pass generating
// the actual response. The response text is the concatenation of both passes.
func TestProvider_Call(t *testing.T) {
	t.Run("text_message", func(t *testing.T) {
		var receivedBodies []string
		srv := newSSEServer(t, []string{
			sseBody("Hello!"),      // 1st pass: tool detection
			sseBody("Hello again"), // 2nd pass: response generation
		}, &receivedBodies)
		defer srv.Close()

		p := NewProvider("Test", srv.URL, WithAPIKey("test-key"))
		resp, err := p.Generate("chat", "gpt-4", Request{
			NewMessage("text/plain", []byte("Hi there")),
		})
		require.NoError(t, err)
		require.Len(t, resp, 1)

		assert.True(t, resp[0].IsText())
		assert.Equal(t, "Hello!Hello again", resp[0].String())

		// The request body must contain the sent user message.
		require.Len(t, receivedBodies, 2)
		for _, body := range receivedBodies {
			assert.Contains(t, body, "Hi there")
			assert.Contains(t, body, "gpt-4")
		}

		// Only the first pass carries the tool definitions.
		assert.Contains(t, receivedBodies[0], "native_fetch")
		assert.NotContains(t, receivedBodies[1], "native_fetch")
	})

	t.Run("http_error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
		}))
		defer srv.Close()

		p := NewProvider("Test", srv.URL, WithAPIKey("invalid-key"))
		resp, err := p.Generate("chat", "gpt-4", Request{
			NewMessage("text/plain", []byte("Hi there")),
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "openai: failed to execute chat completion")
		assert.Empty(t, resp)
	})
}
