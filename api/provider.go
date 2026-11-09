package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
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

// BaseURL returns the API url for this provider, with trailing slash (/).
func (p *llmProvider) BaseURL() string {
	return strings.TrimSuffix(p.baseURL, "/") + "/"
}

// Call should send a [Request] and return a [Response].
func (p *llmProvider) Call(model string, req Request) (Response, error) {
	return p.getClient().Send(model, req)
}

// ----------------------------------------------------------------------------

func (p *llmProvider) getClient() Client {
	switch {
	case p.apiType == APITypeOpenAI:
		p.client = NewClient(p.apiKey)

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
func NewClient(apiKey string, options ...ClientOption) Client {
	c := &openAIClient{
		client: openai.NewClient(
			option.WithAPIKey(apiKey),
		),
	}

	for _, option := range options {
		option(c)
	}

	return c
}

// Send should use the underlying client to send a message.
func (c *openAIClient) Send(model string, req Request) (Response, error) {
	msgUnion := []openai.ChatCompletionMessageParamUnion{}
	for _, msg := range req {
		if msg.IsText() {
			msgUnion = append(msgUnion, openai.UserMessage(msg.String()))
		}

		// TODO: uploads, images, audio, etc.
	}

	chat, err := c.client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Messages: msgUnion,
		Model:    model,
	})
	if err != nil {
		return Response{}, fmt.Errorf("openai: chat completion error: %w", err)
	}

	// TODO: chat completion message contains `.Audio`
	// TODO: chat completion with image and other media
	return Response{
		NewMessage("text/plain", []byte(chat.Choices[0].Message.Content)),
	}, nil
}
