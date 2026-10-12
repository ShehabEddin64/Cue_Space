package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeGemini emulates the native API. behave decides, per model, what a
// generateContent call returns.
func fakeGemini(t *testing.T, behave func(model string, body map[string]any) (int, string)) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-goog-api-key") != "test-key" || r.URL.Query().Get("key") != "" {
			w.WriteHeader(400)
			io.WriteString(w, `{"error":{"code":400,"message":"API key not valid. Please pass a valid API key.","status":"INVALID_ARGUMENT"}}`)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/models")
		switch {
		case r.Method == http.MethodGet && path == "":
			if r.URL.Query().Get("pageToken") == "" {
				io.WriteString(w, `{"models":[
				  {"name":"models/gemini-2.5-flash","supportedGenerationMethods":["generateContent","countTokens"]},
				  {"name":"models/gemini-embedding-001","supportedGenerationMethods":["embedContent"]}],
				 "nextPageToken":"p2"}`)
			} else {
				io.WriteString(w, `{"models":[
				  {"name":"models/gemini-3.5-flash","supportedGenerationMethods":["generateContent"]},
				  {"name":"models/gemini-2.5-flash-preview-tts","supportedGenerationMethods":["generateContent"]}]}`)
			}
		case r.Method == http.MethodGet:
			io.WriteString(w, `{"name":"models`+path+`"}`)
		case r.Method == http.MethodPost && strings.HasSuffix(path, ":generateContent"):
			model := strings.TrimSuffix(strings.TrimPrefix(path, "/"), ":generateContent")
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			status, out := behave(model, body)
			w.WriteHeader(status)
			io.WriteString(w, out)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func answer(text string) string {
	b, _ := json.Marshal(map[string]any{"candidates": []any{map[string]any{
		"content": map[string]any{"role": "model", "parts": []any{map[string]any{"text": text}}}, "finishReason": "STOP",
	}}})
	return string(b)
}

func newTestGemini(srv *httptest.Server, model string, backups ...string) *Gemini {
	g := NewGemini("test-key", model)
	g.base = srv.URL
	g.fallbacks = backups
	return g
}

func TestGeminiProposes(t *testing.T) {
	var sawSchema bool
	srv := fakeGemini(t, func(model string, body map[string]any) (int, string) {
		cfg := body["generationConfig"].(map[string]any)
		sawSchema = cfg["responseJsonSchema"] != nil && cfg["responseMimeType"] == "application/json"
		if body["systemInstruction"] == nil {
			t.Error("system instruction missing")
		}
		return 200, answer(atlasAnswer)
	})
	g := newTestGemini(srv, "gemini-2.5-flash")
	prop, err := Propose(context.Background(), g, "atlas", nil, installed)
	if err != nil {
		t.Fatal(err)
	}
	if prop.Cue.Name != "Atlas" || !sawSchema || prop.Model != "" {
		t.Errorf("name %q, schema %v, model note %q", prop.Cue.Name, sawSchema, prop.Model)
	}
}

func TestGeminiFallsBackToJSONModeWhenSchemaRejected(t *testing.T) {
	calls := 0
	srv := fakeGemini(t, func(model string, body map[string]any) (int, string) {
		calls++
		if body["generationConfig"].(map[string]any)["responseJsonSchema"] != nil {
			return 400, `{"error":{"code":400,"message":"Invalid JSON payload received. Unknown name \"responseJsonSchema\"","status":"INVALID_ARGUMENT"}}`
		}
		return 200, answer("```json\n" + atlasAnswer + "\n```")
	})
	prop, err := Propose(context.Background(), newTestGemini(srv, "gemini-2.5-flash"), "atlas", nil, installed)
	if err != nil || prop.Cue.Name != "Atlas" || calls != 2 {
		t.Fatalf("prop %+v, err %v, calls %d", prop.Cue.Name, err, calls)
	}
}

func TestGeminiBusyModelUsesBackup(t *testing.T) {
	fastRetries(t)
	srv := fakeGemini(t, func(model string, body map[string]any) (int, string) {
		if model == "gemini-3.5-flash" {
			return 503, geminiBusy
		}
		return 200, answer(atlasAnswer)
	})
	prop, err := Propose(context.Background(), newTestGemini(srv, "gemini-3.5-flash", "gemini-2.5-flash"), "atlas", nil, installed)
	if err != nil {
		t.Fatal(err)
	}
	if prop.Model != "Gemini 2.5 Flash" {
		t.Errorf("model note = %q", prop.Model)
	}
}

func TestGeminiPlainErrors(t *testing.T) {
	srv := fakeGemini(t, func(model string, body map[string]any) (int, string) {
		return 200, `{"promptFeedback":{"blockReason":"SAFETY"}}`
	})
	_, err := newTestGemini(srv, "gemini-2.5-flash").Complete(context.Background(), Request{Schema: draftSchema})
	var e *Error
	if !errors.As(err, &e) || e.Kind != KindRefused || strings.Contains(err.Error(), "SAFETY") {
		t.Errorf("blocked: %v", err)
	}

	bad := NewGemini("wrong-key", "gemini-2.5-flash")
	bad.base = srv.URL
	if err := bad.Check(context.Background()); !errors.As(err, &e) || e.Kind != KindAuth {
		t.Errorf("bad key: %v", err)
	}
}

func TestGeminiModels(t *testing.T) {
	srv := fakeGemini(t, nil)
	ids, err := newTestGemini(srv, "gemini-2.5-flash").Models(context.Background())
	if err != nil || strings.Join(ids, ",") != "gemini-2.5-flash,gemini-3.5-flash" {
		t.Errorf("models = %q, %v", ids, err)
	}
	if err := newTestGemini(srv, "gemini-2.5-flash").Check(context.Background()); err != nil {
		t.Errorf("Check: %v", err)
	}
}

func TestNewGeminiUsesNativeAPIWithBackups(t *testing.T) {
	p, err := New(Config{Provider: ProviderGemini, Model: "models/gemini-3.5-flash"}, "k")
	if err != nil {
		t.Fatal(err)
	}
	g, ok := p.(*Gemini)
	if !ok || g.model != "gemini-3.5-flash" || strings.Join(g.fallbacks, ",") != "gemini-2.5-flash,gemini-2.5-flash-lite" {
		t.Errorf("provider = %#v", p)
	}
}
