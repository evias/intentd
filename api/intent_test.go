package api

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testIntentConfig returns a shared [IntentConfig] fixture.
func testIntentConfig() IntentConfig {
	return IntentConfig{
		Name:        "Test Intent",
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
