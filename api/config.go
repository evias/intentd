package api

import (
	"unicode"
	"unicode/utf8"
)

// IntentConfig describes an intent configuration file which defines the
// configuration fields, input fields and system prompt of an intent.
//
// YAML configuration files must contain: name, prompt.
// YAML configuration files may also contain a description.
type IntentConfig struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Fields      map[string]FieldConfig `yaml:"fields"`
	Prompt      string                 `yaml:"prompt"`
}

// FieldConfig describes configuration fields that may be required for the
// execution of the prompt template, e.g. a "url" field for a "feed" intent.
//
// Supported field types include:
// - `binary` is `[]byte`.
// - `string` is `string`
// - `number` is one of `float64`, `uint64`, or `int`
// - `duration` is a string representation of a duration, e.g. `3s`.
// - `attachment` is a file path to a file that will be *uploaded*.
type FieldConfig struct {
	Type       string `yaml:"type"`
	Default    any    `yaml:"default"`
	Required   bool   `yaml:"required"`
	AllowEmpty bool   `yaml:"allowEmpty"`
}

// UpperFirstFieldName returns a field name with the first character uppercased.
func UpperFirstFieldName(s string) string {
	if s == "" {
		return s
	}
	r, i := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[i:]
}
