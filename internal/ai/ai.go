// Package ai turns a plain-language description into a proposed Cue. The AI
// only proposes: callers show the proposal for review and save it themselves.
// Nothing here can run a Cue or write to disk.
package ai

import (
	"context"
	"errors"
	"net/url"
	"strings"
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
	// Models lists the model ids this key can use, for the Settings picker.
	Models(ctx context.Context) ([]string, error)
}

// ErrNotConfigured means the user has not set up an AI provider yet.
var ErrNotConfigured = errors.New("no AI provider configured: add one in Settings")

// Config selects and configures a provider.
type Config struct {
	Provider string `json:"provider"` // "anthropic" | "gemini" | "openai"
	Model    string `json:"model"`
	BaseURL  string `json:"baseUrl"`
}

const (
	ProviderAnthropic = "anthropic"
	ProviderGemini    = "gemini"
	ProviderOpenAI    = "openai"

	DefaultAnthropicModel = "claude-opus-5-5"
	DefaultGeminiModel    = "gemini-2.5-flash"

	// GeminiBaseURL is Google's OpenAI-compatible endpoint for the Gemini API.
	GeminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
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
	case ProviderGemini:
		if apiKey == "" {
			return nil, errors.New("add your Gemini API key in Settings (free at aistudio.google.com)")
		}
		model := cfg.Model
		if model == "" {
			model = DefaultGeminiModel
		}
		return NewOpenAICompat(GeminiBaseURL, apiKey, model), nil
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

// Disclosure says in plain words which model handles requests and where, so
// the UI can be honest about what leaves the Mac.
type Disclosure struct {
	// Model is a display name, e.g. "Claude Opus 5.5" or "llama3.2".
	Model string `json:"model"`
	// Vendor is who runs it, e.g. "Anthropic", "Ollama", "OpenRouter".
	Vendor string `json:"vendor"`
	// Local is true when requests never leave this Mac.
	Local bool `json:"local"`
	// Note is anything else worth knowing about where requests go.
	Note string `json:"note,omitempty"`
}

func Describe(cfg Config) Disclosure {
	switch cfg.Provider {
	case ProviderAnthropic:
		model := cfg.Model
		if model == "" {
			model = DefaultAnthropicModel
		}
		return Disclosure{Model: prettyModel(model), Vendor: "Anthropic"}
	case ProviderGemini:
		model := cfg.Model
		if model == "" {
			model = DefaultGeminiModel
		}
		return Disclosure{
			Model:  prettyModel(strings.TrimPrefix(model, "models/")),
			Vendor: "Google",
			Note:   "On Google’s free tier, Google may use your requests to improve its products.",
		}
	case ProviderOpenAI:
		d := Disclosure{Model: cfg.Model}
		u, err := url.Parse(cfg.BaseURL)
		if err != nil || u.Host == "" {
			d.Vendor = cfg.BaseURL
			return d
		}
		host := u.Hostname()
		d.Local = host == "localhost" || host == "127.0.0.1" || host == "::1"
		switch {
		case d.Local && u.Port() == "11434":
			d.Vendor = "Ollama"
		case d.Local && u.Port() == "1234":
			d.Vendor = "LM Studio"
		case strings.HasSuffix(host, "openai.com"):
			d.Vendor = "OpenAI"
		case strings.HasSuffix(host, "openrouter.ai"):
			d.Vendor = "OpenRouter"
		default:
			d.Vendor = host
		}
		return d
	}
	return Disclosure{}
}

// prettyModel turns "claude-opus-5-5" into "Claude Opus 5.5".
func prettyModel(id string) string {
	parts := strings.Split(id, "-")
	var words, version []string
	for _, p := range parts {
		if p == "" {
			continue
		}
		if p[0] >= '0' && p[0] <= '9' {
			version = append(version, p)
			continue
		}
		if len(version) > 0 { // a word after the version, e.g. a suffix
			words = append(words, strings.Join(version, "."))
			version = nil
		}
		words = append(words, strings.ToUpper(p[:1])+p[1:])
	}
	if len(version) > 0 {
		words = append(words, strings.Join(version, "."))
	}
	return strings.Join(words, " ")
}
