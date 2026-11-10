package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"github.com/evias/intentd/api/tools"
)

type APIType string

const (
	APITypeOpenAI APIType = "openai"
	// TODO: APITypeOllama APIType = "ollama"
)

// Request is a slice of [Streamable], i.e. "messages".
type Request []Streamable

// Response is a slice of [Streamable], i.e. "messages".
type Response []Streamable

// Provider defines the contract for model providers.
type Provider interface {
	// Name should return the provider's name.
	Name() string
	// Type should return a APIType.
	Type() APIType
	// BaseURL should return the API url for this provider.
	BaseURL() string

	// Call should send a [Request] to model and return a [Response].
	Call(model string, req Request) (Response, error)
	// TODO: CallWithOptions(temperature, model, etc.)
}

// ProviderOption defines the contract for [Provider] option helpers.
type ProviderOption func(Provider)

// ----------------------------------------------------------------------------

// llmProvider is a private implementation of the [Provider] interface.
type llmProvider struct {
	name    string
	apiKey  string
	apiType APIType

	client  Client
	baseURL string
}

// Ensure that our implementation satisfies interface.
var _ Provider = (*llmProvider)(nil)

// NewProvider creates a [Provider] of type t with name and baseURL.
func NewProvider(name, baseURL string, options ...ProviderOption) Provider {
	p := &llmProvider{
		name:    name,
		baseURL: baseURL,
		apiType: APITypeOpenAI,
	}

	for _, option := range options {
		option(p)
	}

	return p
}

// WithAPIType implements an option helper to inject an apiType.
func WithAPIType(apiType APIType) ProviderOption {
	return func(p Provider) {
		lp := p.(*llmProvider)
		lp.apiType = apiType
	}
}

// WithAPIKey implements an option helper to inject an apiKey.
func WithAPIKey(apiKey string) ProviderOption {
	return func(p Provider) {
		lp := p.(*llmProvider)
		lp.apiKey = apiKey
	}
}

// Name returns the provider's name.
func (p *llmProvider) Name() string {
	return p.name
}

// Type returns the provider's APIType.
func (p *llmProvider) Type() APIType {
	return p.apiType
}

// BaseURL returns the API url for this provider, without trailing slash (/).
func (p *llmProvider) BaseURL() string {
	return strings.TrimSuffix(p.baseURL, "/")
}

// Call should send a [Request] and return a [Response].
func (p *llmProvider) Call(model string, req Request) (Response, error) {
	return p.getClient().Send(model, req)
}

// ----------------------------------------------------------------------------

func (p *llmProvider) getClient() Client {
	switch {
	case p.apiType == APITypeOpenAI:
		p.client = NewClient(p.baseURL, p.apiKey)

	default:
		panic("apiType '" + p.apiType + "' is not yet implemented")
	}

	return p.client
}

// ----------------------------------------------------------------------------

type Client interface {
	Send(model string, req Request) (Response, error)
}

// ClientOption defines the contract for [Client] option helpers.
type ClientOption func(Client)

type openAIClient struct {
	client openai.Client
}

// Ensure that our implementation satisfies interface.
var _ Client = (*openAIClient)(nil)

// TODO: add ollama API type client
func NewClient(baseURL, apiKey string, options ...ClientOption) Client {
	c := &openAIClient{
		client: openai.NewClient(
			option.WithBaseURL(baseURL),
			option.WithAPIKey(apiKey),
			// option.WithMiddleware(debugResponseMiddleware),
		),
	}

	for _, option := range options {
		option(c)
	}

	slog.Debug("OpenAI client created", "baseURL", baseURL)
	return c
}

// Send uses the [openai.Client] to create a Chat Completions API conversation.
// CAUTION: Currently only supports text-based input/output.
//
// This method execute 2 rounds of Chat Completions API calls. The first pass
// permits to detect `native_fetch` tool calls that we need to inject timely,
// and the second pass will generate the actual response to the prompt.
//
// TODO: context should be fed from caller/provider.
// TODO: should permit image/audio/files input.
// TODO: should permit image/audio/files output.
func (c *openAIClient) Send(model string, req Request) (Response, error) {
	msgUnion := []openai.ChatCompletionMessageParamUnion{}
	inTokens := 0
	for _, msg := range req {
		if msg.IsText() {
			msgUnion = append(msgUnion, openai.UserMessage(msg.String()))
			inTokens += len(msg.String())
		}
		// TODO: should permit image/audio/files input.
	}
	slog.Debug("Sending request with Responses API",
		"model", model,
		"len", inTokens)

	apiParams := openai.ChatCompletionNewParams{
		Model:    model,
		Messages: msgUnion,
		Tools: []openai.ChatCompletionToolUnionParam{
			openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
				Name:        "native_fetch",
				Description: openai.String("Fetch a URL"),
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
			}),
		},
	}

	// 1) First pass with Tools will inject a call to [tools.native_fetch].
	firstResponse, sessionMsgs, err := c.executeChatCompletionWithToolCalls(apiParams)
	if err != nil {
		return Response{}, fmt.Errorf("openai: failed to execute chat completion: %w", err)
	}

	// merges the tool call messages before we start the second pass.
	if len(sessionMsgs) > 0 {
		apiParams.Messages = append(apiParams.Messages, sessionMsgs...)
	}

	// 2) Second pass without Tools, should generate response.
	apiParams.Tools = nil // set nil to omit from request
	secondResponse, _, err := c.executeChatCompletionWithToolCalls(apiParams)
	if err != nil {
		return Response{}, fmt.Errorf("openai: failed to execute chat completion: %w", err)
	}

	slog.Debug("Streaming response from Responses API",
		"model", model,
		"len", len(secondResponse))

	// TODO: should permit image/audio/files output.
	return Response{
		NewMessage("text/plain", []byte(firstResponse+secondResponse)),
	}, nil
}

// ----------------------------------------------------------------------------

// debugResponseMiddleware enables a verbose debug mode where the raw response
// from the API is logged on stderr.
func debugResponseMiddleware(req *http.Request, mn option.MiddlewareNext) (*http.Response, error) {
	resp, err := mn(req)
	if err != nil {
		return resp, err
	}
	// clone body for logging (body is a stream, must re-fill it)
	b, _ := io.ReadAll(resp.Body)
	slog.Error("RAW RESPONSE", "status", resp.Status, "ct", resp.Header.Get("Content-Type"), "body", string(b))
	resp.Body = io.NopCloser(bytes.NewReader(b))
	return resp, err
}

// executeChatCompletionWithToolCalls executes a single pass of a chat
// completion streaming request/response, it evaluates potential tool
// calls and injects a call to the "native_fetch" tool if necessary.
func (c *openAIClient) executeChatCompletionWithToolCalls(
	apiParams openai.ChatCompletionNewParams,
) (string, []openai.ChatCompletionMessageParamUnion, error) {
	session := []openai.ChatCompletionMessageParamUnion{}
	stream := c.client.Chat.Completions.NewStreaming(
		context.Background(),
		apiParams,
		option.WithHeader("X-Title", "intentd runtime"),
	)
	defer func() { _ = stream.Close() }()

	var (
		responseText strings.Builder
		accumulator  = openai.ChatCompletionAccumulator{}
	)
	for stream.Next() {
		chunk := stream.Current()
		if !accumulator.AddChunk(chunk) {
			return "", session, fmt.Errorf("openai: failed to accumulate streaming chunk")
		}

		// If we have text, add it to the response.
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			responseText.WriteString(chunk.Choices[0].Delta.Content)
			slog.Debug("Streaming chunk received", "len", len(chunk.Choices[0].Delta.Content))
		}

		if tool, ok := accumulator.JustFinishedToolCall(); ok {
			slog.Debug("Tool call detected",
				"tool", tool.Name,
				"args", tool.Arguments)
		}
	}

	if err := stream.Err(); err != nil {
		return "", session, fmt.Errorf("openai: streaming ended abruptly: %w", err)
	}

	// No function calls means we are done processing this response as well.
	if len(accumulator.Choices) == 0 || len(accumulator.Choices[0].Message.ToolCalls) == 0 {
		return responseText.String(), session, nil
	}

	// Evaluate function calls and whether we need to inject a custom tool call.
	toolCalls := accumulator.Choices[0].Message.ToolCalls
	session = append(session, accumulator.Choices[0].Message.ToParam())

	// Calls internal tool `native_fetch` under the hood.
	toolSession, err := c.processInternalToolCalls(toolCalls)
	if err != nil {
		return "", session, fmt.Errorf("openai: failed to call internal tool: %w", err)
	}
	session = append(session, toolSession...)

	return responseText.String(), session, nil
}

func (c *openAIClient) processInternalToolCalls(
	toolCalls []openai.ChatCompletionMessageToolCallUnion,
) ([]openai.ChatCompletionMessageParamUnion, error) {
	session := []openai.ChatCompletionMessageParamUnion{}
	for _, tc := range toolCalls {
		// Only native_fetch is supported with this method.
		if tc.Function.Name != "native_fetch" {
			return session, fmt.Errorf("openai: unsupported tool call: %s", tc.Function.Name)
		}

		var args struct {
			Url string `json:"url"`
		}
		decoder := json.NewDecoder(strings.NewReader(tc.Function.Arguments))
		decoder.DisallowUnknownFields()
		decoder.Decode(&args)

		// IMPORTANT: This executes a call to the custom tool `native_fetch`
		// as implemented in `api/tools/native_fetch.go`.
		fetchResult, err := tools.KnownToolsFn["native_fetch"](args.Url)
		if err != nil {
			return session, fmt.Errorf("openai: failed to call tool native_fetch: %w", err)
		}

		session = append(session, openai.ToolMessage(fetchResult, tc.ID))
	}

	return session, nil
}
