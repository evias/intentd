package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"github.com/evias/intentd/api/tools"
)

const (
	// This contains a conservative upperbound for the number of consecutive
	// tool calls that may be executed. This forbids the model to exile on a
	// never-ending scraping task when presented with a URL.
	MaxRecursiveToolCallDepth = 3
)

// ----------------------------------------------------------------------------

type Client interface {
	Chat(model string, req Request) (Response, error)
	Image(model string, req Request) (Response, error)
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
			option.WithMiddleware(debugResponseMiddleware),
		),
	}

	for _, option := range options {
		option(c)
	}

	slog.Debug("OpenAI client created", "baseURL", baseURL)
	return c
}

// Chat uses the [openai.Client] to create a Chat Completions API conversation.
//
// This method execute 2 rounds of Chat Completions API calls. The first pass
// permits to detect `native_fetch` tool calls that we need to inject timely,
// and the second pass will generate the actual response to the prompt.
//
// TODO: context should be fed from caller/provider.
func (c *openAIClient) Chat(model string, req Request) (Response, error) {
	// Add messages from req to []openai.ChatCompletionMessageParamUnion.
	msgUnion, inTokens := buildChatCompletionParams(req)
	slog.Debug("Sending request for Chat Completion API",
		"model", model,
		"len", inTokens)

	apiParams := openai.ChatCompletionNewParams{
		Model:    model,
		Messages: msgUnion,
		Tools:    tools.GetInternalTools(),
	}

	// 1) First X passes with Tools will inject calls to internal tools if necessary.
	//
	// IMPORTANT: This will run one or many passes of Chat Completion API roundtrips,
	// to permit executing multi-stage internal tools, e.g. reading a file with URL.
	// IMPORTANT: The responses for tool-call rounds are *discarded* as not final.
	//
	// This loop's break condition evaluates whether sessionMsgs contains tool call
	// messages, it will be empty otherwise. When empty, proceed to final completion.
	var (
		responseText strings.Builder
		sessionMsgs  []openai.ChatCompletionMessageParamUnion
		err          error
	)
	for num := 0; num < MaxRecursiveToolCallDepth && (num == 0 || len(sessionMsgs) > 0); num++ {
		slog.Debug(fmt.Sprintf("Starting completion round #%d", num+1), "model", model)

		_, sessionMsgs, err = c.executeChatCompletionWithToolCalls(apiParams)
		if err != nil {
			return Response{}, fmt.Errorf("openai: failed to execute chat completion: %w", err)
		}

		// merges the tool call messages before we start the second pass.
		if len(sessionMsgs) > 0 {
			apiParams.Messages = append(apiParams.Messages, sessionMsgs...)
		}

		slog.Debug(fmt.Sprintf("Completed completion round #%d", num+1), "model", model)
	}

	slog.Debug("Now executing final completion round", "model", model)

	// 2) Final pass without Tools, should generate response, i.e. answer.
	apiParams.Tools = nil // set nil to omit from request
	finalResponse, _, err := c.executeChatCompletionWithToolCalls(apiParams)
	if err != nil {
		return Response{}, fmt.Errorf("openai: failed to execute chat completion: %w", err)
	}
	responseText.WriteString(finalResponse)

	slog.Debug("Streaming response for Chat Completion API",
		"model", model,
		"len", responseText.Len())

	return Response{
		NewMessage("text/plain", []byte(responseText.String())),
	}, nil
}

// Image uses the [openai.Client] to create a Image Generation API service.
// CAUTION: It generates exactly one image for now.
//
// TODO: context should be fed from caller/provider.
// TODO: supports only one image generation for now.
func (c *openAIClient) Image(model string, req Request) (Response, error) {
	prompt, numTokens := buildPromptFromRequest(req)
	slog.Debug("Sending request for Image Generation API",
		"model", model,
		"len", numTokens)

	// TODO: context should be fed from caller/provider.
	image, err := c.client.Images.Generate(context.Background(), openai.ImageGenerateParams{
		Prompt:         prompt,
		Model:          model,
		N:              openai.Int(1), // TODO: supports only one image generation for now.
		ResponseFormat: openai.ImageGenerateParamsResponseFormatB64JSON,
	}, option.WithHeader("X-Title", "intentd runtime"))
	if err != nil {
		return Response{}, fmt.Errorf("openai: failed to execute image generation: %w", err)
	}

	// "should not happen", but it does. ;)
	if len(image.Data) == 0 {
		return Response{}, fmt.Errorf("openai: image generation failed to send image data")
	}

	imageBz, err := base64.StdEncoding.DecodeString(image.Data[0].B64JSON)
	if err != nil {
		return Response{}, fmt.Errorf("openai: failed to decode image base64: %w", err)
	}

	slog.Debug("Received image for Image Generation API",
		"model", model,
		"len", len(imageBz))

	return Response{
		NewMessage("image/png", imageBz),
	}, nil
}

// ----------------------------------------------------------------------------

// executeChatCompletionWithToolCalls executes a single pass of a chat
// completion streaming request/response, it evaluates potential tool
// calls and injects a call to the "native_fetch" tool if necessary.
//
// Returns a responseText, []ChatCompletionMessageParamUnion, error
func (c *openAIClient) executeChatCompletionWithToolCalls(
	apiParams openai.ChatCompletionNewParams,
) (
	string,
	[]openai.ChatCompletionMessageParamUnion,
	error,
) {
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

// processInternalToolCalls forwards tool call messages with function name
// and arguments, to the underlying internal tool, i.e. native_fetch.
//
// See also: [tools.CallInternalTool]
func (c *openAIClient) processInternalToolCalls(
	toolCalls []openai.ChatCompletionMessageToolCallUnion,
) ([]openai.ChatCompletionMessageParamUnion, error) {
	session := []openai.ChatCompletionMessageParamUnion{}
	for _, tc := range toolCalls {
		// Tool calls for internal tools only
		if !tools.HasInternalTool(tc.Function.Name) {
			return session, fmt.Errorf("openai: unsupported tool call: %s", tc.Function.Name)
		}

		// IMPORTANT: This executes a call to the internal tools, e.g. `native_fetch`
		// as implemented in `api/tools/native_fetch.go`.
		toolResult, err := tools.CallInternalTool(tc.Function.Name, tc.Function.Arguments)
		if err != nil {
			return session, fmt.Errorf("openai: failed to call tool %s: %w", tc.Function.Name, err)
		}

		session = append(session, openai.ToolMessage(toolResult, tc.ID))
	}

	return session, nil
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
	slog.Debug("RAW RESPONSE", "status", resp.Status, "ct", resp.Header.Get("Content-Type"), "body", string(b))
	resp.Body = io.NopCloser(bytes.NewReader(b))
	return resp, err
}

// buildPromptFromRequest reads all *text* messages and concatenates them
// in one prompt.
// CAUTION: This method only supports **text inputs**.
func buildPromptFromRequest(req Request) (
	prompt string,
	numTokens int,
) {
	var promptTpl strings.Builder

	for _, msg := range req {
		if msg.IsText() {
			promptTpl.WriteString(msg.String())
		}
	}

	prompt = promptTpl.String()
	numTokens += len(prompt)
	return // prompt, numTokens
}

// buildChatCompletionParams builds a chat completion message parameters union
// from a [Request], of which each entry may be text, image, audio or file upload.
func buildChatCompletionParams(req Request) (
	inputs []openai.ChatCompletionMessageParamUnion,
	numTokens int,
) {
	inputs = []openai.ChatCompletionMessageParamUnion{}

	// Each message in req may contain text, image, audio or file uploads.
	// Each type of message needs different types for openai SDK.
	for _, msg := range req {
		if msg.IsText() {
			inputs = append(inputs, openai.UserMessage(msg.String()))
		} else if strings.HasPrefix(msg.ContentType(), "image/") {
			inputs = append(inputs, openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{{
				OfImageURL: &openai.ChatCompletionContentPartImageParam{
					ImageURL: openai.ChatCompletionContentPartImageImageURLParam{
						URL: msg.String(),
					},
				},
			}}))
		} else if strings.HasPrefix(msg.ContentType(), "audio/") {
			inputs = append(inputs, openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{{
				OfInputAudio: &openai.ChatCompletionContentPartInputAudioParam{
					InputAudio: openai.ChatCompletionContentPartInputAudioInputAudioParam{
						Data: msg.String(),
					},
				},
			}}))
		} else {
			inputs = append(inputs, openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{{
				OfFile: &openai.ChatCompletionContentPartFileParam{
					File: openai.ChatCompletionContentPartFileFileParam{
						FileData: openai.Opt(msg.String()),
					},
				},
			}}))
		}

		numTokens += len(msg.String())
	}

	return // inputs, numTokens
}
