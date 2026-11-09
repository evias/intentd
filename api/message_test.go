package api

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestMessage_MarshalBinary tests the type+size-prefixed binary encoding.
func TestMessage_MarshalBinary(t *testing.T) {
	t.Run("wire_format", func(t *testing.T) {
		msg := NewMessage("image/png", []byte("payload"))

		b, err := msg.MarshalBinary()
		assert.NoError(t, err)

		// Wire format: [uint16 big-endian type][uint32 big-endian length][payload]
		assert.Len(t, b, PrefixSize+len("payload"))
		assert.Equal(t, uint16(MimeTypeIDImagePng), binary.BigEndian.Uint16(b[0:2]))
		assert.Equal(t, uint32(7), binary.BigEndian.Uint32(b[2:6]))
		assert.Equal(t, []byte("payload"), b[PrefixSize:])
	})

	t.Run("output_length", func(t *testing.T) {
		msg := NewMessage("text/plain", []byte("payload"))

		b, err := msg.MarshalBinary()
		assert.NoError(t, err)
		assert.Len(t, b, PrefixSize+len("payload"))
	})

	t.Run("payload_too_large", func(t *testing.T) {
		msg := NewMessage("text/plain", make([]byte, MaxPayload+1))
		b, err := msg.MarshalBinary()
		assert.Error(t, err)
		assert.Nil(t, b)
	})
}

// marshalFixture returns a handcrafted type+size-prefixed bytes buffer,
// independent of [Message.MarshalBinary].
func marshalFixture(t *testing.T, id MimeTypeID, payload string) []byte {
	t.Helper()
	b := make([]byte, PrefixSize+len(payload))
	binary.BigEndian.PutUint16(b, uint16(id))
	binary.BigEndian.PutUint32(b[2:], uint32(len(payload)))
	copy(b[PrefixSize:], payload)
	return b
}

// TestMessage_UnmarshalBinary tests the decoding of type+size-prefixed bytes.
func TestMessage_UnmarshalBinary(t *testing.T) {
	t.Run("handcrafted_buffer", func(t *testing.T) {
		decoded := &Message{}
		require.NoError(t, decoded.UnmarshalBinary(
			marshalFixture(t, MimeTypeIDImagePng, "payload")))

		assert.Equal(t, "image/png", decoded.ContentType())
		assert.Equal(t, []byte("payload"), decoded.Payload)
	})

	t.Run("empty_payload", func(t *testing.T) {
		decoded := &Message{}
		require.NoError(t, decoded.UnmarshalBinary(
			marshalFixture(t, MimeTypeIDTextPlain, "")))
		assert.Equal(t, "text/plain", decoded.ContentType())
		assert.Empty(t, decoded.Payload)
	})

	t.Run("buffer_too_short", func(t *testing.T) {
		decoded := &Message{}
		err := decoded.UnmarshalBinary(make([]byte, PrefixSize-1))
		assert.ErrorIs(t, err, ErrShort)
	})

	t.Run("declared_length_exceeds_max", func(t *testing.T) {
		b := make([]byte, PrefixSize+4)
		binary.BigEndian.PutUint32(b[2:], MaxPayload+1)

		decoded := &Message{}
		err := decoded.UnmarshalBinary(b)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "declared length")
	})

	t.Run("truncated_payload", func(t *testing.T) {
		b := marshalFixture(t, MimeTypeIDTextPlain, "payload")
		b = b[:len(b)-2] // payload shorter than declared size

		decoded := &Message{}
		err := decoded.UnmarshalBinary(b)
		assert.ErrorIs(t, err, ErrShort)
	})

	t.Run("payload_is_copied", func(t *testing.T) {
		// UnmarshalBinary must not alias the caller's buffer.
		b := marshalFixture(t, MimeTypeIDTextPlain, "payload")

		decoded := &Message{}
		require.NoError(t, decoded.UnmarshalBinary(b))

		b[PrefixSize] = 'P' // mutate the input buffer
		assert.Equal(t, []byte("payload"), decoded.Payload)
	})
}

// TestMessage_ToBytes tests that ToBytes mirrors MarshalBinary.
func TestMessage_ToBytes(t *testing.T) {
	msg := NewMessage("text/plain", []byte("payload"))

	b, err := msg.ToBytes()
	require.NoError(t, err)

	expected, err := msg.MarshalBinary()
	require.NoError(t, err)
	assert.Equal(t, expected, b)
}

// TestLoadMessageFromBytes tests message loading from a bytes slice.
func TestLoadMessageFromBytes(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		msg, err := LoadMessageFromBytes(
			marshalFixture(t, MimeTypeIDImagePng, "payload"))
		require.NoError(t, err)
		require.NotNil(t, msg)
		assert.Equal(t, "image/png", msg.ContentType())
		assert.Equal(t, []byte("payload"), msg.Payload)
	})

	t.Run("invalid", func(t *testing.T) {
		msg, err := LoadMessageFromBytes(make([]byte, PrefixSize-1))
		assert.Error(t, err)
		assert.Nil(t, msg)
	})
}

// TestMessage_WriteTo tests writing the prefixed message to an io.Writer.
func TestMessage_WriteTo(t *testing.T) {
	t.Run("payload_too_large", func(t *testing.T) {
		msg := NewMessage("text/plain", make([]byte, MaxPayload+1))
		n, err := msg.WriteTo(&bytes.Buffer{})
		assert.Error(t, err)
		assert.Zero(t, n)
	})

	t.Run("roundtrip", func(t *testing.T) {
		var buf bytes.Buffer
		msg := NewMessage("image/png", []byte("payload"))

		n, err := msg.WriteTo(&buf)
		require.NoError(t, err)
		assert.Equal(t, int64(len("payload")+PrefixSize), n)

		decoded := &Message{}
		read, err := decoded.ReadFrom(&buf)
		require.NoError(t, err)
		assert.Equal(t, int64(len("payload")+PrefixSize), read)
		assert.Equal(t, msg.ContentType(), decoded.ContentType())
		assert.Equal(t, msg.Payload, decoded.Payload)
	})
}

// TestMessage_ReadFrom tests reading a prefixed message from an io.Reader.
func TestMessage_ReadFrom(t *testing.T) {
	t.Run("handcrafted_buffer", func(t *testing.T) {
		r := bytes.NewReader(marshalFixture(t, MimeTypeIDImagePng, "payload"))

		decoded := &Message{}
		read, err := decoded.ReadFrom(r)
		require.NoError(t, err)
		assert.Equal(t, int64(len("payload")+PrefixSize), read)
		assert.Equal(t, "image/png", decoded.ContentType())
		assert.Equal(t, []byte("payload"), decoded.Payload)
	})

	t.Run("declared_length_exceeds_max", func(t *testing.T) {
		b := make([]byte, PrefixSize)
		binary.BigEndian.PutUint32(b[2:], MaxPayload+1)

		decoded := &Message{}
		read, err := decoded.ReadFrom(bytes.NewReader(b))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "declared length")
		assert.Equal(t, int64(PrefixSize), read)
	})

	t.Run("header_too_short", func(t *testing.T) {
		decoded := &Message{}
		_, err := decoded.ReadFrom(bytes.NewReader(make([]byte, PrefixSize-1)))
		assert.Error(t, err)
	})

	t.Run("payload_too_short", func(t *testing.T) {
		// Header declares a payload that is not fully readable.
		b := marshalFixture(t, MimeTypeIDTextPlain, "payload")
		b = b[:len(b)-2]

		decoded := &Message{}
		read, err := decoded.ReadFrom(bytes.NewReader(b))
		assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
		assert.Equal(t, int64(len("payload")+PrefixSize-2), read)
	})
}
