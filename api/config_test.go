package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestUpperFirstFieldName tests unicode-aware first-rune capitalization.
func TestUpperFirstFieldName(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "lowercase", input: "speed", expected: "Speed"},
		{name: "already_upper", input: "Speed", expected: "Speed"},
		{name: "empty", input: "", expected: ""},
		{name: "single_char", input: "a", expected: "A"},
		{name: "multibyte_rune", input: "ünïcode", expected: "Ünïcode"},
		{name: "digit_prefix", input: "1abc", expected: "1abc"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equalf(t, tc.expected, UpperFirstFieldName(tc.input),
				"expected %q, got %q", tc.expected, UpperFirstFieldName(tc.input))
		})
	}
}
