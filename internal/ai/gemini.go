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
	"time"

	"github.com/ShehabEddin64/Cue_Space/internal/i18n"
)

// GeminiAPI is Google's native Gemini API endpoint.
const GeminiAPI = "https://generativelanguage.googleapis.com/v1beta"

// Gemini talks to Google's native Gemini API (generateContent), which
// supports JSON Schema answers directly.
type Gemini struct {
	base      string
	apiKey    string
	model     string
	fallbacks []string
	used      string
	http      *http.Client
}

func NewGemini(apiKey, model string) *Gemini {
	return &Gemini{
		base:   GeminiAPI,
		apiKey: apiKey,
		model:  strings.TrimPrefix(model, "models/"),
		http:   &http.Client{Timeout: 2 * time.Minute},
	}
}

func (g *Gemini) ConfiguredModel() string { return g.model }
func (g *Gemini) UsedModel() string       { return g.used }

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

// Complete asks for JSON matching req.Schema, retrying busy moments and
// stepping in with backup models, like the other providers.
func (g *Gemini) Complete(ctx context.Context, req Request) (string, error) {
	models := append([]string{g.model}, g.fallbacks...)
	out, used, err := completeWithBackups(ctx, models, func(model string) (string, error) {
		return g.generate(ctx, model, req)
	})
	if err != nil {
		return "", err
	}
	g.used = used
	return out, nil
}

// generate makes one request to one model. It asks for a JSON Schema answer
// first; if the API rejects the schema, it falls back to JSON mode with the
// schema in the prompt.
func (g *Gemini) generate(ctx context.Context, model string, req Request) (string, error) {
	schemaText, _ := json.Marshal(req.Schema)
	configs := []struct {
		system string
		config map[string]any
	}{
		{req.System, map[string]any{"responseMimeType": "application/json", "responseJsonSchema": req.Schema}},
		{req.System + "\n\nAnswer with only a JSON object that matches this JSON Schema:\n" + string(schemaText),
			map[string]any{"responseMimeType": "application/json"}},
	}
	var resp struct {
		Candidates []struct {
			Content      geminiContent `json:"content"`
			FinishReason string        `json:"finishReason"`
		} `json:"candidates"`
		PromptFeedback struct {
			BlockReason string `json:"blockReason"`
		} `json:"promptFeedback"`
	}
	var err error
	for _, c := range configs {
		body := map[string]any{
			"systemInstruction": geminiContent{Parts: []geminiPart{{Text: c.system}}},
			"contents":          []geminiContent{{Role: "user", Parts: []geminiPart{{Text: req.User}}}},
			"generationConfig":  c.config,
		}
		err = g.do(ctx, http.MethodPost, "/models/"+url.PathEscape(model)+":generateContent", model, body, &resp)
		var e *Error
		if !errors.As(err, &e) || e.Kind != KindBadRequest {
			break // success, or a failure a simpler request won't fix
		}
	}
	if err != nil {
		return "", err
	}
	if resp.PromptFeedback.BlockReason != "" {
		return "", &Error{Kind: KindRefused, Vendor: "Gemini", Model: model, Detail: "blocked: " + resp.PromptFeedback.BlockReason}
	}
	if len(resp.Candidates) == 0 {
		return "", &Error{Kind: KindOther, Vendor: "Gemini", Model: model, Detail: "no candidates"}
	}
	cand := resp.Candidates[0]
	switch cand.FinishReason {
	case "MAX_TOKENS":
		return "", i18n.Errorf("The answer was cut off. Try a shorter description.")
	case "SAFETY", "BLOCKLIST", "PROHIBITED_CONTENT", "SPII", "RECITATION":
		return "", &Error{Kind: KindRefused, Vendor: "Gemini", Model: model, Detail: "finish: " + cand.FinishReason}
	}
	var text strings.Builder
	for _, p := range cand.Content.Parts {
		text.WriteString(p.Text)
	}
	return text.String(), nil
}

// Check looks up the model, which validates the key without generating.
func (g *Gemini) Check(ctx context.Context) error {
	var m struct {
		Name string `json:"name"`
	}
	_, err := withRetry(ctx, 1, func() (string, error) {
		return "", g.do(ctx, http.MethodGet, "/models/"+url.PathEscape(g.model), g.model, nil, &m)
	})
	return err
}

// Models lists the models this key can use to generate text.
func (g *Gemini) Models(ctx context.Context) ([]string, error) {
	var ids []string
	token := ""
	for page := 0; page < 10; page++ {
		var resp struct {
			Models []struct {
				Name    string   `json:"name"`
				Methods []string `json:"supportedGenerationMethods"`
			} `json:"models"`
			NextPageToken string `json:"nextPageToken"`
		}
		path := "/models?pageSize=1000"
		if token != "" {
			path += "&pageToken=" + url.QueryEscape(token)
		}
		_, err := withRetry(ctx, 1, func() (string, error) {
			return "", g.do(ctx, http.MethodGet, path, "", nil, &resp)
		})
		if err != nil {
			return nil, err
		}
		for _, m := range resp.Models {
			id := strings.TrimPrefix(m.Name, "models/")
			if !strings.HasPrefix(id, "gemini") || containsAny(id, "embedding", "image", "tts", "audio", "live", "computer") {
				continue
			}
			for _, method := range m.Methods {
				if method == "generateContent" {
					ids = append(ids, id)
					break
				}
			}
		}
		if resp.NextPageToken == "" {
			break
		}
		token = resp.NextPageToken
	}
	sort.Strings(ids)
	return ids, nil
}

func (g *Gemini) do(ctx context.Context, method, path, model string, in, out any) error {
	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, g.base+path, body)
	if err != nil {
		return &Error{Kind: KindOther, Vendor: "Gemini", Detail: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", g.apiKey) // a header, so the key never lands in a URL or log
	resp, err := g.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return &Error{Kind: KindNetwork, Vendor: "Gemini", Model: model, Detail: err.Error()}
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return &Error{Kind: KindNetwork, Vendor: "Gemini", Model: model, Detail: err.Error()}
	}
	if resp.StatusCode >= 300 {
		return classifyHTTP(resp.StatusCode, string(data), "Gemini", model)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return &Error{Kind: KindOther, Vendor: "Gemini", Model: model, Detail: "unreadable response: " + err.Error()}
	}
	return nil
}
