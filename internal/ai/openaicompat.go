package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/ShehabEddin64/Cue_Space/internal/i18n"
)

// OpenAICompat talks to any server that implements the OpenAI chat
// completions API: OpenAI, Gemini, OpenRouter, Ollama, LM Studio, and others.
type OpenAICompat struct {
	baseURL string
	apiKey  string
	model   string
	vendor  string
	local   bool
	// fallbacks are tried, in order, when the chosen model stays busy or is
	// out of quota. Empty for most servers.
	fallbacks []string
	used      string
	http      *http.Client
}

func NewOpenAICompat(baseURL, apiKey, model string) *OpenAICompat {
	base := strings.TrimRight(baseURL, "/")
	vendor, local := vendorOf(base)
	return &OpenAICompat{
		baseURL: base,
		apiKey:  apiKey,
		model:   model,
		vendor:  vendor,
		local:   local,
		// Local models can be slow on first load.
		http: &http.Client{Timeout: 5 * time.Minute},
	}
}

// vendorOf names the service behind a base URL for messages.
func vendorOf(base string) (name string, local bool) {
	u, err := url.Parse(base)
	if err != nil || u.Hostname() == "" {
		return "The AI server", false
	}
	host := u.Hostname()
	local = host == "localhost" || host == "127.0.0.1" || host == "::1"
	switch {
	case strings.Contains(host, "generativelanguage.googleapis.com") || strings.Contains(base, "generativelanguage.googleapis.com"):
		return "Gemini", false
	case strings.HasSuffix(host, "openai.com"):
		return "OpenAI", false
	case strings.HasSuffix(host, "openrouter.ai"):
		return "OpenRouter", false
	case local && u.Port() == "11434":
		return "Ollama", true
	case local && u.Port() == "1234":
		return "LM Studio", true
	case local:
		return "Your local AI server", true
	}
	return host, false
}

// UsedModel is the model that answered the last Complete call; it differs
// from the configured one when a backup model stepped in.
func (o *OpenAICompat) UsedModel() string { return o.used }

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Complete asks for JSON matching req.Schema. Temporary failures are retried
// quietly; if the chosen model stays busy (or is out of quota), backup
// models are tried before giving up with a plain-language error.
func (o *OpenAICompat) Complete(ctx context.Context, req Request) (string, error) {
	models := append([]string{o.model}, o.fallbacks...)
	out, used, err := completeWithBackups(ctx, models, func(model string) (string, error) {
		return o.completeWith(ctx, model, req)
	})
	if err != nil {
		return "", err
	}
	o.used = used
	return out, nil
}

// ConfiguredModel is the model chosen in Settings.
func (o *OpenAICompat) ConfiguredModel() string { return o.model }

// completeWith makes one request to one model. Servers differ in how much of
// OpenAI's structured output they support, so it steps down: a strict JSON
// schema, then plain JSON mode with the schema in the prompt, then no
// response format at all (the answer is still parsed as JSON).
func (o *OpenAICompat) completeWith(ctx context.Context, model string, req Request) (string, error) {
	schemaText, _ := json.Marshal(req.Schema)
	withSchemaInPrompt := req.System + "\n\nAnswer with only a JSON object that matches this JSON Schema:\n" + string(schemaText)
	formats := []struct {
		system string
		format map[string]any
	}{
		{req.System, map[string]any{
			"type":        "json_schema",
			"json_schema": map[string]any{"name": req.SchemaName, "strict": true, "schema": req.Schema},
		}},
		{withSchemaInPrompt, map[string]any{"type": "json_object"}},
		{withSchemaInPrompt, nil},
	}
	var resp struct {
		Choices []struct {
			Message      chatMessage `json:"message"`
			FinishReason string      `json:"finish_reason"`
		} `json:"choices"`
	}
	var err error
	for _, f := range formats {
		body := map[string]any{
			"model": model,
			"messages": []chatMessage{
				{Role: "system", Content: f.system},
				{Role: "user", Content: req.User},
			},
		}
		if f.format != nil {
			body["response_format"] = f.format
		}
		err = o.do(ctx, http.MethodPost, "/chat/completions", model, body, &resp)
		var e *Error
		if !errors.As(err, &e) || e.Kind != KindBadRequest {
			break // success, or a failure a simpler request won't fix
		}
	}
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", &Error{Kind: KindOther, Vendor: o.vendor, Model: model, Detail: "no choices in response"}
	}
	if resp.Choices[0].FinishReason == "length" {
		return "", i18n.Errorf("The answer was cut off. Try a shorter description.")
	}
	return resp.Choices[0].Message.Content, nil
}

// Check lists models, which validates the URL and key without generating.
func (o *OpenAICompat) Check(ctx context.Context) error {
	ids, err := o.listModels(ctx)
	if err != nil {
		return err
	}
	want := strings.TrimPrefix(o.model, "models/")
	for _, id := range ids {
		if id == want {
			return nil
		}
	}
	if len(ids) > 0 {
		return &Error{Kind: KindModel, Vendor: o.vendor, Model: o.model}
	}
	return nil
}

// Models lists the server's models. For Gemini, only text models that can
// chat are kept.
func (o *OpenAICompat) Models(ctx context.Context) ([]string, error) {
	ids, err := o.listModels(ctx)
	if err != nil {
		return nil, err
	}
	if o.vendor != "Gemini" {
		return ids, nil
	}
	var chat []string
	for _, id := range ids {
		if strings.HasPrefix(id, "gemini") && !containsAny(id, "embedding", "image", "tts", "audio", "live", "veo", "imagen", "computer") {
			chat = append(chat, id)
		}
	}
	return chat, nil
}

func (o *OpenAICompat) listModels(ctx context.Context) ([]string, error) {
	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_, err := withRetry(ctx, 1, func() (string, error) {
		return "", o.do(ctx, http.MethodGet, "/models", "", nil, &resp)
	})
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(resp.Data))
	for _, m := range resp.Data {
		ids = append(ids, strings.TrimPrefix(m.ID, "models/")) // Gemini says "models/gemini-…"
	}
	sort.Strings(ids)
	return ids, nil
}

func containsAny(s string, parts ...string) bool {
	for _, p := range parts {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

func (o *OpenAICompat) do(ctx context.Context, method, path, model string, in, out any) error {
	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, o.baseURL+path, body)
	if err != nil {
		return &Error{Kind: KindOther, Vendor: o.vendor, Detail: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	if o.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+o.apiKey)
	}
	resp, err := o.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		e := &Error{Kind: KindNetwork, Vendor: o.vendor, Model: model, Detail: err.Error()}
		if o.local && errors.Is(err, syscall.ECONNREFUSED) {
			e.Kind = KindLocalOff
		}
		return e
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return &Error{Kind: KindNetwork, Vendor: o.vendor, Model: model, Detail: err.Error()}
	}
	if resp.StatusCode >= 300 {
		e := classifyHTTP(resp.StatusCode, string(data), o.vendor, model)
		if ra := resp.Header.Get("Retry-After"); ra != "" && e.RetryAfter == 0 {
			if d, err := time.ParseDuration(ra + "s"); err == nil {
				e.RetryAfter = d
			}
		}
		return e
	}
	if err := json.Unmarshal(data, out); err != nil {
		return &Error{Kind: KindOther, Vendor: o.vendor, Model: model, Detail: "unreadable response: " + err.Error()}
	}
	return nil
}
