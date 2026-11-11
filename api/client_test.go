package api

import (
	"encoding/json"
	"testing"

	"github.com/openai/openai-go/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildPromptFromRequest tests prompt building from *text* messages.
func TestBuildPromptFromRequest(t *testing.T) {
	t.Run("text_messages_only", func(t *testing.T) {
		req := Request{
			NewMessage("text/plain", []byte("Hello, ")),
			NewMessage("text/plain", []byte("world")),
		}

		prompt, numTokens := buildPromptFromRequest(req)
		assert.Equal(t, "Hello, world", prompt)
		assert.Equal(t, len("Hello, world"), numTokens)
	})

	t.Run("non_text_messages_are_ignored", func(t *testing.T) {
		req := Request{
			NewMessage("text/plain", []byte("Hello")),
			NewMessage("image/png", []byte("not-a-prompt")),
		}

		prompt, numTokens := buildPromptFromRequest(req)
		assert.Equal(t, "Hello", prompt)
		assert.Equal(t, len("Hello"), numTokens)
	})
}

// TestBuildChatCompletionParams tests chat completion parameter building from
// mixed content-type [Request] messages.
func TestBuildChatCompletionParams(t *testing.T) {
	marshal := func(t *testing.T, inputs []openai.ChatCompletionMessageParamUnion) []string {
		t.Helper()
		encoded := make([]string, len(inputs))
		for i, input := range inputs {
			bz, err := json.Marshal(input)
			require.NoError(t, err)
			encoded[i] = string(bz)
		}
		return encoded
	}

	t.Run("text_message", func(t *testing.T) {
		inputs, numTokens := buildChatCompletionParams(Request{
			NewMessage("text/plain", []byte("Hi there")),
		})
		require.Len(t, inputs, 1)
		assert.Equal(t, len("Hi there"), numTokens)

		encoded := marshal(t, inputs)
		assert.Contains(t, encoded[0], `"Hi there"`)
		assert.Contains(t, encoded[0], `"role":"user"`)
	})

	t.Run("image_message", func(t *testing.T) {
		inputs, _ := buildChatCompletionParams(Request{
			NewMessage("image/png", []byte("https://evi.as/image.png")),
		})
		require.Len(t, inputs, 1)

		encoded := marshal(t, inputs)
		assert.Contains(t, encoded[0], "https://evi.as/image.png")
		assert.Contains(t, encoded[0], "image_url")
	})

	t.Run("audio_message", func(t *testing.T) {
		inputs, _ := buildChatCompletionParams(Request{
			NewMessage("audio/wav", []byte("audio-data")),
		})
		require.Len(t, inputs, 1)

		encoded := marshal(t, inputs)
		assert.Contains(t, encoded[0], "audio-data")
		assert.Contains(t, encoded[0], "input_audio")
	})

	t.Run("file_message", func(t *testing.T) {
		// Any other content type is treated as a file upload.
		inputs, _ := buildChatCompletionParams(Request{
			NewMessage("application/pdf", []byte("file-data")),
		})
		require.Len(t, inputs, 1)

		encoded := marshal(t, inputs)
		assert.Contains(t, encoded[0], "file-data")
		assert.Contains(t, encoded[0], "file")
	})

	t.Run("num_tokens_counts_all_messages", func(t *testing.T) {
		_, numTokens := buildChatCompletionParams(Request{
			NewMessage("text/plain", []byte("Hello")),           // 5 bytes
			NewMessage("image/png", []byte("0123456789")),       // 10 bytes
			NewMessage("application/pdf", []byte("0123456789")), // 10 bytes
		})
		assert.Equal(t, 25, numTokens)
	})
}
