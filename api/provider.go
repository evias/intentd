package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
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
		),
	}

	for _, option := range options {
		option(c)
	}

	return c
}

// Send uses the [openai.Client] to create a Responses API conversation.
// CAUTION: Currently only support text-based input/output.
//
// TODO: context should be fed from caller/provider.
// TODO: should permit image/audio/files input.
// TODO: should permit image/audio/files output.
func (c *openAIClient) Send(model string, req Request) (Response, error) {
	msgUnion := responses.ResponseNewParamsInputUnion{}
	for _, msg := range req {
		if msg.IsText() {
			msgUnion.OfString = openai.String(msg.String())
		}
		// TODO: should permit image/audio/files input.
	}

	// TODO: context should be fed from caller/provider.
	resp, err := c.client.Responses.New(context.Background(), responses.ResponseNewParams{
		Input: msgUnion,
		Model: model,
	}, option.WithHeader("HTTP-Referer", "https://ai.dhealth.com"),
		option.WithHeader("X-Title", "AIDH Bots"))
	if err != nil {
		return Response{}, fmt.Errorf("openai: chat completion error: %w", err)
	}

	// TODO: should permit image/audio/files output.
	return Response{
		NewMessage("text/plain", []byte(resp.OutputText())),
	}, nil
}
