package tools

import "github.com/openai/openai-go/v3"

type ToolFunc func(string) (string, error)

var KnownTools = []string{
	"native_fetch",
}

var KnownToolsFn = map[string]ToolFunc{
	"native_fetch": native_fetch,
}

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
