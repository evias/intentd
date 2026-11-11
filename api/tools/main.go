package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/openai/openai-go/v3"
)

const (
	ToolCallMaxRuntimeSeconds = 10 * time.Second
)

// ToolFunc defines the signature for internal tool closures.
type ToolFunc func(context.Context, string) (string, error)

// ToolResult defines a wrapper for tool call results and errors.
type ToolResult struct {
	Value string
	Err   error
}

// ToolCallConfig contains configuration fields for individual tool calls.
type ToolCallConfig struct {
	MaxRuntimeSeconds time.Duration
}

var DefaultToolCallConfig = ToolCallConfig{
	MaxRuntimeSeconds: ToolCallMaxRuntimeSeconds,
}

// Closures mapping.
var internalToolsByName = map[string]ToolFunc{
	"native_fetch": native_fetch,
}

// GetInternalTools returns a list of internal tools enabled with this library.
func GetInternalTools() []openai.ChatCompletionToolUnionParam {
	tools := []openai.ChatCompletionToolUnionParam{}
	tools = append(tools, openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
		Name:        "native_fetch",
		Description: openai.String("Internal URL fetch tool"),
		Strict:      openai.Bool(true),
		Parameters: openai.FunctionParameters{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"url": map[string]string{
					"type": "string",
				},
			},
			"required": []string{"url"},
		},
	}))
	return tools
}

// CallInternalTool executes an internal tool call by name.
// The internal tool is executed in a spawned goroutine.
func CallInternalTool(
	name string,
	args string,
) (string, error) {
	return CallInternalToolWithConfig(name, args, DefaultToolCallConfig)
}

// CallInternalToolCallInternalToolWithConfig executes an internal tool
// call by name.
// The internal tool is executed in a spawned goroutine.
func CallInternalToolWithConfig(
	name string,
	args string,
	conf ToolCallConfig,
) (string, error) {
	if _, ok := internalToolsByName[name]; !ok {
		return "", fmt.Errorf("tools: internal tool %s is not supported", name)
	}

	ctx, cancelFn := context.WithTimeout(context.Background(), conf.MaxRuntimeSeconds)
	defer cancelFn()

	// Spawns a goroutine for the internal tool call
	resultCh := make(chan ToolResult, 1)
	go func() {
		// IMPORTANT: ctx is forwarded to http.Get.
		result, err := internalToolsByName[name](ctx, args)
		resultCh <- ToolResult{result, err}
	}()

	// Make sure we don't exceed ToolCallMaxRuntimeSeconds.
	var result ToolResult
	select {
	case result = <-resultCh:
	case <-ctx.Done():
		result = ToolResult{
			Value: "",
			Err:   fmt.Errorf("tools: internal tool %s timed out after 10 seconds", name),
		}
	}

	return result.Value, result.Err
}
