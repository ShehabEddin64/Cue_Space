package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"syscall"
	"time"
)

// OpenAICompat talks to any server that implements the OpenAI chat
// completions API: OpenAI, OpenRouter, Ollama (http://localhost:11434/v1),
// LM Studio (http://localhost:1234/v1), and others.
type OpenAICompat struct {
	baseURL string
	apiKey  string
	model   string
	http    *http.Client
}

func NewOpenAICompat(baseURL, apiKey, model string) *OpenAICompat {
	return &OpenAICompat{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		// Local models can be slow on first load.
		http: &http.Client{Timeout: 5 * time.Minute},
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Complete asks for JSON matching req.Schema. Servers differ in how much of
// OpenAI's structured output they support, so it steps down: a strict JSON
// schema, then plain JSON mode with the schema in the prompt, then no
// response format at all (the answer is still parsed as JSON).
func (o *OpenAICompat) Complete(ctx context.Context, req Request) (string, error) {
	schemaText, _ := json.Marshal(req.Schema)
	withSchemaInPrompt := req.System + "\n\nAnswer with only a JSON object that matches this JSON Schema:\n" + string(schemaText)
	attempts := []struct {
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
	for _, a := range attempts {
		body := map[string]any{
			"model": o.model,
			"messages": []chatMessage{
				{Role: "system", Content: a.system},
				{Role: "user", Content: req.User},
			},
		}
		if a.format != nil {
			body["response_format"] = a.format
		}
		err = o.do(ctx, http.MethodPost, "/chat/completions", body, &resp)
		var bad *badRequest
		if !errors.As(err, &bad) {
			break // success, or a failure a simpler request won't fix
		}
	}
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", errors.New("the model returned no answer")
	}
	if resp.Choices[0].FinishReason == "length" {
		return "", errors.New("the model's answer was cut off; try a shorter description")
	}
	return resp.Choices[0].Message.Content, nil
}

// Check lists models, which validates the URL and key without generating.
func (o *OpenAICompat) Check(ctx context.Context) error {
	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := o.do(ctx, http.MethodGet, "/models", nil, &resp); err != nil {
		return err
	}
	for _, m := range resp.Data {
		// Gemini lists ids as "models/gemini-…".
		if m.ID == o.model || strings.TrimPrefix(m.ID, "models/") == strings.TrimPrefix(o.model, "models/") {
			return nil
		}
	}
	if len(resp.Data) > 0 {
		return fmt.Errorf("connected, but model %q is not available on this server", o.model)
	}
	return nil
}

// Models lists the server's models. For Gemini, only text models that can
// chat are kept.
func (o *OpenAICompat) Models(ctx context.Context) ([]string, error) {
	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := o.do(ctx, http.MethodGet, "/models", nil, &resp); err != nil {
		return nil, err
	}
	gemini := strings.Contains(o.baseURL, "generativelanguage.googleapis.com")
	var ids []string
	for _, m := range resp.Data {
		id := strings.TrimPrefix(m.ID, "models/")
		if gemini && (!strings.HasPrefix(id, "gemini") || containsAny(id, "embedding", "image", "tts", "audio", "live", "veo", "imagen", "computer")) {
			continue
		}
		ids = append(ids, id)
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

// badRequest is a 400/404/422 answer: the server didn't accept the request
// as sent, so a simpler request may work.
type badRequest struct{ msg string }

func (e *badRequest) Error() string { return e.msg }

func (o *OpenAICompat) do(ctx context.Context, method, path string, in, out any) error {
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
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if o.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+o.apiKey)
	}
	resp, err := o.http.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		if errors.Is(err, syscall.ECONNREFUSED) {
			return fmt.Errorf("nothing is answering at %s. Is the server running?", o.baseURL)
		}
		return fmt.Errorf("could not reach %s: %w", o.baseURL, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		msg := fmt.Sprintf("%s returned %d: %s", o.baseURL, resp.StatusCode, apiErrorMessage(data))
		switch resp.StatusCode {
		case http.StatusBadRequest, http.StatusUnprocessableEntity:
			return &badRequest{msg}
		case http.StatusUnauthorized, http.StatusForbidden:
			return fmt.Errorf("the server rejected the API key (%d): %s", resp.StatusCode, apiErrorMessage(data))
		case http.StatusTooManyRequests:
			return fmt.Errorf("rate limited; free tiers allow only a few requests per minute. Try again shortly")
		}
		return errors.New(msg)
	}
	return json.Unmarshal(data, out)
}

func apiErrorMessage(data []byte) string {
	var e struct {
		Error any `json:"error"`
	}
	if json.Unmarshal(data, &e) == nil && e.Error != nil {
		switch v := e.Error.(type) {
		case string:
			return v
		case map[string]any:
			if m, ok := v["message"].(string); ok {
				return m
			}
		}
	}
	s := strings.TrimSpace(string(data))
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}
