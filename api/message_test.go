package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewMessage tests [Message] construction from a content type.
func TestNewMessage(t *testing.T) {
	t.Run("known_content_type", func(t *testing.T) {
		msg := NewMessage("image/png", []byte("payload"))
		assert.Equal(t, "image/png", msg.ContentType())
		assert.Equal(t, []byte("payload"), msg.Payload)
	})

	t.Run("unknown_content_type", func(t *testing.T) {
		// Unknown mime types default to application/octet-stream.
		msg := NewMessage("not/a-mime-type", []byte("payload"))
		assert.Equal(t, "application/octet-stream", msg.ContentType())
		assert.Equal(t, []byte("payload"), msg.Payload)
	})

	t.Run("empty_payload", func(t *testing.T) {
		msg := NewMessage("text/plain", []byte{})
		assert.Equal(t, "text/plain", msg.ContentType())
		assert.Empty(t, msg.Payload)
	})
}

// TestMessage_IsText tests text content type detection.
func TestMessage_IsText(t *testing.T) {
	testCases := []struct {
		name         string
		contentType  string
		expectedText bool
	}{
		{name: "text_plain", contentType: "text/plain", expectedText: true},
		{name: "image", contentType: "image/png", expectedText: false},
		{name: "octet_stream", contentType: "application/octet-stream", expectedText: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := NewMessage(tc.contentType, []byte("payload"))
			assert.Equalf(t, tc.expectedText, msg.IsText(),
				"expected IsText()=%v for %q", tc.expectedText, tc.contentType)
		})
	}
}

// TestMessage_String tests the string representation of the message payload.
func TestMessage_String(t *testing.T) {
	msg := NewMessage("text/plain", []byte("hello, world"))
	assert.Equal(t, "hello, world", msg.String())

	msg = NewMessage("text/plain", []byte{})
	assert.Equal(t, "", msg.String())
}
