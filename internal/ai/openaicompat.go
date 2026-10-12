package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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

func (o *OpenAICompat) Complete(ctx context.Context, req Request) (string, error) {
	body := map[string]any{
		"model": o.model,
		"messages": []chatMessage{
			{Role: "system", Content: req.System},
			{Role: "user", Content: req.User},
		},
		"response_format": map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   req.SchemaName,
				"strict": true,
				"schema": req.Schema,
			},
		},
	}
	var resp struct {
		Choices []struct {
			Message      chatMessage `json:"message"`
			FinishReason string      `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := o.do(ctx, http.MethodPost, "/chat/completions", body, &resp); err != nil {
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
		if m.ID == o.model {
			return nil
		}
	}
	if len(resp.Data) > 0 {
		return fmt.Errorf("connected, but model %q is not available on this server", o.model)
	}
	return nil
}

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
		return fmt.Errorf("%s returned %d: %s", o.baseURL, resp.StatusCode, apiErrorMessage(data))
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
