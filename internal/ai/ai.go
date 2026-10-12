// Package ai turns a plain-language description into a proposed Cue. The AI
// only proposes: callers show the proposal for review and save it themselves.
// Nothing here can run a Cue or write to disk.
package ai

import (
	"context"
	"errors"
)

// Request is one structured completion: the model must answer with JSON that
// matches Schema.
type Request struct {
	System     string
	User       string
	SchemaName string
	Schema     map[string]any
}

// Provider is a model the user brought: Anthropic, or any OpenAI-compatible
// endpoint (OpenAI, OpenRouter, Ollama, LM Studio, …).
type Provider interface {
	// Complete returns the model's JSON answer as text.
	Complete(ctx context.Context, req Request) (string, error)
	// Check verifies the key and model without spending tokens where possible.
	Check(ctx context.Context) error
}

// ErrNotConfigured means the user has not set up an AI provider yet.
var ErrNotConfigured = errors.New("no AI provider configured: add one in Settings")

// Config selects and configures a provider.
type Config struct {
	Provider string `json:"provider"` // "anthropic" | "openai"
	Model    string `json:"model"`
	BaseURL  string `json:"baseUrl"`
}

const (
	ProviderAnthropic = "anthropic"
	ProviderOpenAI    = "openai"

	DefaultAnthropicModel = "claude-opus-5-5"
)

// New builds the provider described by cfg. apiKey may be empty for local
// OpenAI-compatible servers such as Ollama.
func New(cfg Config, apiKey string) (Provider, error) {
	switch cfg.Provider {
	case ProviderAnthropic:
		if apiKey == "" {
			return nil, errors.New("add your Anthropic API key in Settings")
		}
		model := cfg.Model
		if model == "" {
			model = DefaultAnthropicModel
		}
		return NewAnthropic(apiKey, model, cfg.BaseURL), nil
	case ProviderOpenAI:
		if cfg.BaseURL == "" {
			return nil, errors.New("set the base URL of your OpenAI-compatible server in Settings")
		}
		if cfg.Model == "" {
			return nil, errors.New("choose a model in Settings")
		}
		return NewOpenAICompat(cfg.BaseURL, apiKey, cfg.Model), nil
	case "":
		return nil, ErrNotConfigured
	}
	return nil, errors.New("unknown AI provider " + cfg.Provider)
}
