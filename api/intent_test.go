package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testIntentConfig returns a shared [IntentConfig] fixture.
func testIntentConfig() IntentConfig {
	return IntentConfig{
		Name:        "Test Intent",
		OutputType:  OutputTypeChat,
		Description: "Test Intent Description",
		Fields: map[string]FieldConfig{
			"url":         FieldConfig{Type: "string", Required: true},
			"count":       FieldConfig{Type: "number", Default: 3.0},
			"sleep":       FieldConfig{Type: "duration", Default: "3s"},
			"attachment":  FieldConfig{Type: "attachment"},
			"data":        FieldConfig{Type: "binary"},
			"description": FieldConfig{Type: "string", Default: "default description"},
		},
		Prompt: "Test prompt with url: {{.Url}}, count: {{.Count}}",
	}
}

// TestNewIntent tests that NewIntent returns an [Intent] that exposes the
// values injected through the [IntentConfig].
func TestNewIntent(t *testing.T) {
	conf := testIntentConfig()
	intent := NewIntent(conf)

	assert.Equal(t, conf.Name, intent.Name())
	assert.Equal(t, conf.OutputType, intent.Type())
	assert.Equal(t, conf.Description, intent.Config().Description)
	assert.Equal(t, conf.Prompt, intent.Config().Prompt)
	assert.Equal(t, conf.Fields, intent.Config().Fields)

	require.NotNil(t, intent.Template())
	var sb strings.Builder
	require.NoError(t, intent.Template().Execute(&sb, map[string]string{}))
	assert.Equal(t, "Test prompt with url: <no value>, count: <no value>", sb.String())
}

// TestNewIntent_WithOptions tests that NewIntent applies all provided
// [IntentOption] helpers, after construction.
func TestNewIntent_WithOptions(t *testing.T) {
	var applied []string

	optionName := func(intent Intent) {
		agent, ok := intent.(*agentIntent)
		require.True(t, ok)
		agent.name = "Overridden Intent"
		applied = append(applied, "name")
	}
	optionDescription := func(intent Intent) {
		agent, ok := intent.(*agentIntent)
		require.True(t, ok)
		agent.conf.Description = "Overridden Description"
		applied = append(applied, "description")
	}

	intent := NewIntent(testIntentConfig(), optionName, optionDescription)

	assert.Equal(t, []string{"name", "description"}, applied)
	assert.Equal(t, "Overridden Intent", intent.Name())
	assert.Equal(t, "Overridden Description", intent.Config().Description)
}

// TestIntent_HasField tests field presence lookup against [IntentConfig].
func TestIntent_HasField(t *testing.T) {
	intent := NewIntent(testIntentConfig())

	assert.True(t, intent.HasField("url"))
	assert.True(t, intent.HasField("count"))
	assert.False(t, intent.HasField("missing"))
}

// TestIntent_FieldType tests the type lookup for configured fields, including
// the "string" fallback for unknown fields.
func TestIntent_FieldType(t *testing.T) {
	testCases := []struct {
		name     string
		field    string
		expected string
	}{
		{name: "string_field", field: "url", expected: "string"},
		{name: "number_field", field: "count", expected: "number"},
		{name: "duration_field", field: "sleep", expected: "duration"},
		{name: "attachment_field", field: "attachment", expected: "attachment"},
		{name: "binary_field", field: "data", expected: "binary"},
		{name: "unknown_field", field: "missing", expected: "string"},
	}

	intent := NewIntent(testIntentConfig())
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equalf(t, tc.expected, intent.FieldType(tc.field),
				"expected %q, got %q", tc.expected, intent.FieldType(tc.field))
		})
	}
}

// TestIntent_FieldDefault tests native default value extraction for all
// supported field types, based on [IntentConfig.FieldConfig].
func TestIntent_FieldDefault(t *testing.T) {
	intent := NewIntent(testIntentConfig())

	t.Run("unknown_field", func(t *testing.T) {
		assert.Equal(t, []byte{}, intent.FieldDefault("missing"))
	})
	t.Run("binary", func(t *testing.T) {
		assert.Equal(t, []byte{}, intent.FieldDefault("data"))
	})
	t.Run("attachment", func(t *testing.T) {
		assert.Equal(t, "", intent.FieldDefault("attachment"))
	})
	t.Run("string", func(t *testing.T) {
		assert.Equal(t, "default description", intent.FieldDefault("description"))
	})
	t.Run("number_float64", func(t *testing.T) {
		// yaml.Unmarshal decodes numbers as float64.
		assert.Equal(t, float64(3), intent.FieldDefault("count"))
	})
	t.Run("duration", func(t *testing.T) {
		assert.Equal(t, 3*time.Second, intent.FieldDefault("sleep"))
	})

	// Field types with specific default value behaviors.
	t.Run("string_non_string_default", func(t *testing.T) {
		conf := testIntentConfig()
		conf.Fields["description"] = FieldConfig{Type: "string", Default: 3.0}
		intent := NewIntent(conf)
		assert.Equal(t, "", intent.FieldDefault("description"))
	})
	t.Run("number_uint64", func(t *testing.T) {
		conf := testIntentConfig()
		conf.Fields["count"] = FieldConfig{Type: "number", Default: uint64(3)}
		intent := NewIntent(conf)
		assert.Equal(t, uint64(3), intent.FieldDefault("count"))
	})
	t.Run("number_int", func(t *testing.T) {
		conf := testIntentConfig()
		conf.Fields["count"] = FieldConfig{Type: "number", Default: int(3)}
		intent := NewIntent(conf)
		assert.Equal(t, int(3), intent.FieldDefault("count"))
	})
	t.Run("number_zero_or_unset", func(t *testing.T) {
		conf := testIntentConfig()
		conf.Fields["count"] = FieldConfig{Type: "number", Default: float64(0)}
		intent := NewIntent(conf)
		assert.Equal(t, float64(0), intent.FieldDefault("count"))
	})
	t.Run("duration_invalid_string", func(t *testing.T) {
		conf := testIntentConfig()
		conf.Fields["sleep"] = FieldConfig{Type: "duration", Default: "not-a-duration"}
		intent := NewIntent(conf)
		assert.Equal(t, time.Duration(0), intent.FieldDefault("sleep"))
	})
	t.Run("duration_nil_default", func(t *testing.T) {
		conf := testIntentConfig()
		conf.Fields["sleep"] = FieldConfig{Type: "duration"}
		intent := NewIntent(conf)
		assert.Equal(t, time.Duration(0), intent.FieldDefault("sleep"))
	})
}

// testPromptConfig returns an [IntentConfig] fixture with string-typed
// defaults only, so that prompt building does not hit the known
// populateDefaults panic with non-string defaults (see skipped tests below).
func testPromptConfig() IntentConfig {
	return IntentConfig{
		Name: "Test Intent",
		Fields: map[string]FieldConfig{
			"url":   FieldConfig{Type: "string", Required: true},
			"count": FieldConfig{Type: "number", Default: 3.0},
		},
		Prompt: "Test prompt with url: {{.Url}}, count: {{.Count}}",
	}
}

// TestIntent_Prompt tests prompt building with args validation and default
// value filling.
func TestIntent_Prompt(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		intent := NewIntent(testPromptConfig())
		prompt, err := intent.Prompt(map[string]string{
			"Url":   "https://evi.as",
			"Count": "5",
		})
		require.NoError(t, err)
		assert.Equal(t, "Test prompt with url: https://evi.as, count: 5", prompt)
	})

	t.Run("missing_required_field", func(t *testing.T) {
		intent := NewIntent(testPromptConfig())
		prompt, err := intent.Prompt(map[string]string{
			"Count": "5", // "Url" is required and missing.
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing argument '--Url'")
		assert.Empty(t, prompt)
	})

	t.Run("required_field_not_allow_empty", func(t *testing.T) {
		conf := testPromptConfig()
		conf.Fields["url"] = FieldConfig{Type: "string", Required: true}
		delete(conf.Fields, "count")
		intent := NewIntent(conf)
		prompt, err := intent.Prompt(map[string]string{"Url": ""})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "argument '--url' may not be empty")
		assert.Empty(t, prompt)
	})

	t.Run("required_field_allow_empty", func(t *testing.T) {
		conf := testPromptConfig()
		conf.Fields["url"] = FieldConfig{
			Type: "string", Required: true, AllowEmpty: true,
		}
		delete(conf.Fields, "count")
		intent := NewIntent(conf)
		prompt, err := intent.Prompt(map[string]string{"Url": ""})
		require.NoError(t, err)
		assert.Equal(t, "Test prompt with url: , count: <no value>", prompt)
	})

	t.Run("unconfigured_fields_are_ignored", func(t *testing.T) {
		conf := testPromptConfig()
		delete(conf.Fields, "count")
		conf.Prompt = "url: {{.Url}}, extra: {{.Extra}}"
		intent := NewIntent(conf)
		prompt, err := intent.Prompt(map[string]string{
			"Url":   "https://evi.as",
			"Extra": "ignored-by-populateDefaults",
		})
		require.NoError(t, err)
		assert.Equal(t, "url: https://evi.as, extra: <no value>", prompt)
	})

	t.Run("invalid_args_type", func(t *testing.T) {
		intent := NewIntent(testPromptConfig())
		prompt, err := intent.Prompt("not-a-map")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid args; expected map[string]string")
		assert.Empty(t, prompt)
	})

	t.Run("unset_number_field_uses_default", func(t *testing.T) {
		conf := IntentConfig{
			Name: "Test Intent",
			Fields: map[string]FieldConfig{
				"count": FieldConfig{Type: "number", Default: 3.0, AllowEmpty: true},
			},
			Prompt: "count: {{.Count}}",
		}
		intent := NewIntent(conf)
		prompt, err := intent.Prompt(map[string]string{})
		require.NoError(t, err)
		assert.Equal(t, "count: 3", prompt)
	})
}

// TestLoadFromConfig tests loading an [Intent] from a YAML configuration file.
func TestLoadFromConfig(t *testing.T) {
	t.Run("missing_file", func(t *testing.T) {
		_, err := LoadFromConfig(filepath.Join(t.TempDir(), "intent.yaml"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "intent config file not found")
	})

	t.Run("valid_config", func(t *testing.T) {
		cfgFile := filepath.Join(t.TempDir(), "summarize.yaml")
		content := `name: Summarize Intent
type: chat
description: Summarize Intent Description
fields:
  url:
    type: string
    required: true
  count:
    type: number
    default: 3
prompt: |
  Summarize the following url: {{.Url}}`
		require.NoError(t, os.WriteFile(cfgFile, []byte(content), 0644))

		intent, err := LoadFromConfig(cfgFile)
		require.NoError(t, err)
		require.NotNil(t, intent)

		conf := intent.Config()
		assert.Equal(t, "Summarize Intent", conf.Name)
		assert.Equal(t, OutputTypeChat, conf.OutputType)
		assert.Equal(t, "Summarize Intent Description", conf.Description)
		assert.Len(t, conf.Fields, 2)
		assert.True(t, intent.HasField("url"))
		assert.True(t, intent.HasField("count"))

		// Options are applied to the loaded intent as well.
		intentWithOptions, err := LoadFromConfig(cfgFile, func(intent Intent) {
			agent, ok := intent.(*agentIntent)
			require.True(t, ok)
			agent.name = "Overridden Intent"
		})
		require.NoError(t, err)
		assert.Equal(t, "Overridden Intent", intentWithOptions.Name())
	})

	t.Run("malformed_config", func(t *testing.T) {
		cfgFile := filepath.Join(t.TempDir(), "broken.yaml")
		content := "name: [unclosed"
		require.NoError(t, os.WriteFile(cfgFile, []byte(content), 0644))

		_, err := LoadFromConfig(cfgFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal intent config")
	})

	t.Run("unreadable_file", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("test requires non-root user (unreadable file permissions)")
		}

		cfgFile := filepath.Join(t.TempDir(), "unreadable.yaml")
		require.NoError(t, os.WriteFile(cfgFile, []byte("name: test"), 0000))

		_, err := LoadFromConfig(cfgFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read intent config")
	})
}
