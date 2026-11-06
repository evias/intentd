package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/evias/intentd/api"
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
			assert.Equalf(t, tc.expected, api.UpperFirstFieldName(tc.input),
				"expected %q, got %q", tc.expected, api.UpperFirstFieldName(tc.input))
		})
	}
}

// TestSliceToMap tests bash-style argument parsing.
func TestSliceToMap(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		expected map[string]string
	}{
		{
			name:     "space_separated",
			args:     []string{"--speed", "10"},
			expected: map[string]string{"--speed": "10"},
		},
		{
			name:     "equals_sign_trailing",
			args:     []string{"--speed=10"},
			expected: map[string]string{"--speed": "10"},
		},
		{
			name:     "flag_without_value",
			args:     []string{"--debug"},
			expected: map[string]string{"--debug": ""},
		},
		{
			name:     "separator_is_skipped",
			args:     []string{"--", "--speed", "10"},
			expected: map[string]string{"--speed": "10"},
		},
		{
			name:     "multiple_flags",
			args:     []string{"--count", "10", "--limit", "30"},
			expected: map[string]string{"--count": "10", "--limit": "30"},
		},
		{
			name:     "consecutive_flags",
			args:     []string{"--a", "--b", "5"},
			expected: map[string]string{"--a": "", "--b": "5"},
		},
		{
			name:     "equals_sign_before_flag_value",
			args:     []string{"--a=1", "--b"},
			expected: map[string]string{"--a": "1", "--b": ""},
		},
		{
			name:     "empty",
			args:     []string{},
			expected: map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equalf(t, tc.expected, sliceToMap(tc.args),
				"expected %v, got %v", tc.expected, sliceToMap(tc.args))
		})
	}
}

// testParserIntent provides a [api.Intent] fixture with fields and params.
func testParserIntent() api.Intent {
	return api.NewIntent(api.IntentConfig{
		Name:        "Parser Intent",
		Description: "Parser Intent Description",
		Fields: map[string]api.FieldConfig{
			"Url":   api.FieldConfig{Type: "string"},
			"Count": api.FieldConfig{Type: "number"},
		},
		Prompt: "Parser test prompt with url: {{.Url}} and count: {{.Count}}",
	})
}

// TestParseIntentArgs tests data argument parsing for command fields and params.
func TestParseIntentArgs(t *testing.T) {
	intent := testParserIntent()

	data := ParseIntentArgs(intent, []string{
		"--url", "https://evi.as", "--count", "3",
	})

	assert.Equal(t, map[string]string{
		"Url":   "https://evi.as",
		"Count": "3",
	}, data)
}

// TestParseIntentArgs_UnknownFields tests that absent or unknown fields are
// ignored.
func TestParseIntentArgs_UnknownParams(t *testing.T) {
	intent := testParserIntent()

	// Unknown field name: ignored.
	data := ParseIntentArgs(intent, []string{"--unknown", "value"})
	assert.NotContains(t, data, "Unknown")
}
