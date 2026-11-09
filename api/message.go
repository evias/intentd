//go:generate go run github.com/evias/intentd/cmd/mimegen -url https://raw.githubusercontent.com/jshttp/mime-db/release-1-54-0/db.json -pkg api -o mimeids_gen.go
package api

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	MaxPayload = 1 << 24 // 16 Mb sanity cap for now
	PrefixSize = 2 + 4   // 2 bytes mime type id, 4 bytes payload size
)

// Streamable defines the contract for structures that may be transported
// in the form of type+size-prefixed bytes payloads.
type Streamable interface {
	// ContentType should return the underlying content type, e.g. "image/png".
	ContentType() string
	// IsText should return true for text messages.
	IsText() bool

	// ToBytes should encode to a bytes slice prefixed with its size and content type.
	ToBytes() ([]byte, error)
	// String should return the string representation of the message.
	String() string

	// WriteTo should write the type+size-prefixed message to a [io.Writer].
	WriteTo(w io.Writer) (int64, error)
	// ReadFrom should read a type+size-prefixed message from a [io.Reader].
	ReadFrom(r io.Reader) (int64, error)
}

// Message defines a message which contains a **type and size**-prefixed
// bytes slice to represent the message. This is notably useful when handling
// non-text LLM responses such as images and/or audio file types.
type Message struct {
	msgType uint16 // 2 bytes
	msgSize uint32 // 4 bytes
	// does NOT contain the type and size prefixes.
	Payload []byte // x bytes
}

// Ensure that our implementation satisfies interface [Streamable].
var _ Streamable = (*Message)(nil)

// NewMessage creates a [Message] of type ct from a byte slice bz.
// This function expects ct to contain a valid mime type, e.g. "image/png".
func NewMessage(ct string, bz []byte) *Message {
	// Parse content type or default to application/octet-stream.
	msgType, ok := MimeTypeIDFromString(ct)
	if !ok {
		// default is for raw []byte payload
		msgType = MimeTypeIDApplicationOctetStream
	}

	return &Message{
		Payload: bz[:],
		msgType: uint16(msgType),
	}
}

// ContentType should return the underlying content type, e.g. "image/png".
func (m *Message) ContentType() string {
	return MimeTypeID(m.msgType).String()
}

// IsText returns true for text messages
func (m *Message) IsText() bool {
	ct := m.ContentType()
	return strings.HasPrefix(ct, "text/")
}

// ToBytes encodes the [Message#Payload] to a bytes slice.
// ToBytes implements Wireable
func (m *Message) ToBytes() ([]byte, error) { return m.MarshalBinary() }

// String returns a string representation of the [Message#Payload].
func (m *Message) String() string {
	return string(m.Payload)
}

// ----------------------------------------------------------------------------

var ErrShort = errors.New("msg: buffer too short")

// MarshalBinary encodes the message to a size-prefixed bytes slice or returns
// an error. It prepends the payload size to the payload bytes as a Uint32.
// Wire format: [uint32 big-endian length][payload]
// MarshalBinary implements encoding.BinaryMarshaler
func (m *Message) MarshalBinary() ([]byte, error) {
	if len(m.Payload) > MaxPayload {
		return nil, fmt.Errorf("msg: payload too large (%d)", len(m.Payload))
	}
	out := make([]byte, PrefixSize+len(m.Payload)) // type+size+bytes
	binary.BigEndian.PutUint16(out, uint16(m.msgType))
	binary.BigEndian.PutUint32(out, uint32(len(m.Payload)))
	copy(out[PrefixSize:], m.Payload)
	return out, nil
}

// UnmarshalBinary decodes a size-prefixed bytes payload to a [Message#Payload]
// or returns an error. It expects at least 4 bytes (the size prefix as uint32),
// and it enforces a MaxPayload size of 1Mb (`1 << 20`).
// UnmarshalBinary implements encoding.BinaryUnmarshaler
func (m *Message) UnmarshalBinary(b []byte) error {
	if len(b) < PrefixSize {
		return ErrShort
	}
	t := binary.BigEndian.Uint16(b)
	n := binary.BigEndian.Uint32(b[2:])
	if n > MaxPayload {
		return fmt.Errorf("msg: declared length %d exceeds max", n)
	}
	if uint32(len(b)-PrefixSize) < n {
		return ErrShort
	}
	m.msgType = t
	m.Payload = append([]byte(nil), b[PrefixSize:PrefixSize+n]...) // copy: don't alias caller's buffer
	return nil
}

// ----------------------------------------------------------------------------

// WriteTo may be used to write directly to socket connections or other transports.
// WriteTo implements io.WriterTo
func (m *Message) WriteTo(w io.Writer) (int64, error) {
	b, err := m.MarshalBinary()
	if err != nil {
		return 0, err
	}
	n, err := w.Write(b)
	return int64(n), err
}

// ReadFrom may be used to read directly from socket connections or other transports.
// ReadFrom implements io.ReaderFrom
func (m *Message) ReadFrom(r io.Reader) (int64, error) {
	var hdr [PrefixSize]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return 0, err
	}
	t := binary.BigEndian.Uint16(hdr[:])
	n := binary.BigEndian.Uint32(hdr[2:])
	if n > MaxPayload {
		return PrefixSize, fmt.Errorf("msg: declared length %d exceeds max", n)
	}
	m.msgType = t
	m.Payload = make([]byte, n)
	read, err := io.ReadFull(r, m.Payload)
	return PrefixSize + int64(read), err
}

// ----------------------------------------------------------------------------

// LoadMessageFromBytes loads a [Message] instance from a bytes slice payload.
func LoadMessageFromBytes(b []byte) (*Message, error) {
	var m Message
	if err := m.UnmarshalBinary(b); err != nil {
		return nil, err
	}
	return &m, nil
}
