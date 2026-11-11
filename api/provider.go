package api

import (
	"strings"
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

	// Generate should send a [Request] to model and return a [Response].
	Generate(t OutputType, model string, req Request) (Response, error)
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

// Generate evaluates the [OutputType], one of "chat" or "image", and calls
// the appropriate client method to process [Request] and return a [Response].
func (p *llmProvider) Generate(t OutputType, model string, req Request) (Response, error) {
	switch {
	case t == OutputTypeImage:
		return p.getClient().Image(model, req)
	default:
	}

	return p.getClient().Chat(model, req)
}

// ----------------------------------------------------------------------------

func (p *llmProvider) getClient() Client {
	if p.client == nil {
		switch {
		case p.apiType == APITypeOpenAI:
			p.client = NewClient(p.baseURL, p.apiKey)

		default:
			panic("apiType '" + p.apiType + "' is not yet implemented")
		}
	}

	return p.client
}
