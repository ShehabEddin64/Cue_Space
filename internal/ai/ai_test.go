package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ShehabEddin64/Cue_Space/internal/cue"
	"github.com/ShehabEddin64/Cue_Space/internal/platform"
)

var installed = []platform.App{
	{Name: "Claude", BundleID: "com.anthropic.claudefordesktop"},
	{Name: "Cursor", BundleID: "com.todesktop.230313mzl4w4u92"},
	{Name: "Safari", BundleID: "com.apple.Safari"},
}

// A recorded answer for "Create a Cue for working on Atlas. I need my GitHub
// repo, IDE, Claude, localhost and documentation."
const atlasAnswer = `{
  "name": "Atlas", "icon": "🛰️", "description": "Work on the Atlas project",
  "notes": "localhost assumed to be port 3000.",
  "resources": [
    {"type":"url","label":"GitHub repo","url":null,"appName":null,"appBundleId":null,"path":null,"openWithName":null,"openWithBundleId":null,"needsInput":"What is the URL of the Atlas repository?"},
    {"type":"folder","label":"Atlas code","url":null,"appName":null,"appBundleId":null,"path":null,"openWithName":"Cursor","openWithBundleId":"com.todesktop.230313mzl4w4u92","needsInput":"Where is the Atlas folder on your Mac?"},
    {"type":"app","label":"Claude","url":null,"appName":"Claude","appBundleId":"com.anthropic.claudefordesktop","path":null,"openWithName":null,"openWithBundleId":null,"needsInput":null},
    {"type":"url","label":"localhost","url":"http://localhost:3000","appName":null,"appBundleId":null,"path":null,"openWithName":null,"openWithBundleId":null,"needsInput":null},
    {"type":"app","label":"Figma","url":null,"appName":"Figma","appBundleId":"com.figma.Desktop","path":null,"openWithName":null,"openWithBundleId":null,"needsInput":null},
    {"type":"shell","label":"npm run dev","url":null,"appName":null,"appBundleId":null,"path":null,"openWithName":null,"openWithBundleId":null,"needsInput":null}
  ]
}`

type fakeProvider struct {
	answer string
	got    Request
}

func (f *fakeProvider) Complete(_ context.Context, r Request) (string, error) {
	f.got = r
	return f.answer, nil
}
func (f *fakeProvider) Check(context.Context) error { return nil }

func TestProposeAtlas(t *testing.T) {
	p := &fakeProvider{answer: atlasAnswer}
	prop, err := Propose(context.Background(), p, "Create a Cue for working on Atlas", nil, installed)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(p.got.System, "Cursor — com.todesktop.230313mzl4w4u92") {
		t.Error("installed apps missing from the prompt")
	}
	c := prop.Cue
	if c.Name != "Atlas" || c.ID == "" {
		t.Errorf("cue = %+v", c)
	}
	// The unknown "shell" type is dropped, never executed.
	if len(c.Resources) != 5 {
		t.Fatalf("got %d resources", len(c.Resources))
	}
	if c.Resources[1].OpenWith.BundleID != "com.todesktop.230313mzl4w4u92" {
		t.Errorf("openWith = %+v", c.Resources[1].OpenWith)
	}
	// Two questions from the model plus one for the app that isn't installed.
	if len(prop.Questions) != 3 {
		t.Errorf("questions = %v", prop.Questions)
	}
	if q := prop.Questions[c.Resources[4].ID]; !strings.Contains(q, "Figma") {
		t.Errorf("missing-app question = %q", q)
	}
	// The proposal is not valid until the user answers the questions, which is
	// what keeps the review step mandatory.
	if c.Validate() == nil {
		t.Error("expected the proposal to need input before saving")
	}
}

func TestProposeEditKeepsIdentity(t *testing.T) {
	existing := &cue.Cue{ID: "keep-me", Name: "Math", Resources: []cue.Resource{{ID: "r", Type: cue.TypeURL, URL: "https://a.b"}}}
	answer := `{"name":"Math","icon":"📐","description":"","notes":"","resources":[
	  {"type":"url","label":"Course","url":"https://a.b","appName":null,"appBundleId":null,"path":null,"openWithName":null,"openWithBundleId":null,"needsInput":null},
	  {"type":"app","label":"Calculator","url":null,"appName":"calculator","appBundleId":null,"path":null,"openWithName":null,"openWithBundleId":null,"needsInput":null}]}`
	apps := append(installed, platform.App{Name: "Calculator", BundleID: "com.apple.calculator"})
	p := &fakeProvider{answer: "```json\n" + answer + "\n```"}
	prop, err := Propose(context.Background(), p, "add the calculator", existing, apps)
	if err != nil {
		t.Fatal(err)
	}
	if prop.Cue.ID != "keep-me" {
		t.Errorf("id = %q", prop.Cue.ID)
	}
	if !strings.Contains(p.got.User, `"https://a.b"`) {
		t.Error("existing cue not sent to the model")
	}
	if got := prop.Cue.Resources[1].App; got.BundleID != "com.apple.calculator" || got.Name != "Calculator" {
		t.Errorf("app matched by name = %+v", got)
	}
	if err := prop.Cue.Validate(); err != nil {
		t.Error(err)
	}
}

func TestProposeRejectsGarbage(t *testing.T) {
	for _, answer := range []string{"Sure! Here is your cue.", `{"name":""}`} {
		if _, err := Propose(context.Background(), &fakeProvider{answer: answer}, "x", nil, installed); err == nil {
			t.Errorf("expected an error for %q", answer)
		}
	}
}

func TestOpenAICompat(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			io.WriteString(w, `{"data":[{"id":"llama3.2"}]}`)
		case "/v1/chat/completions":
			if r.Header.Get("Authorization") != "" {
				t.Error("no key should mean no Authorization header")
			}
			json.NewDecoder(r.Body).Decode(&body)
			json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{
				"message": map[string]any{"role": "assistant", "content": atlasAnswer}, "finish_reason": "stop",
			}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	p, err := New(Config{Provider: ProviderOpenAI, BaseURL: srv.URL + "/v1/", Model: "llama3.2"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Check(context.Background()); err != nil {
		t.Error(err)
	}
	prop, err := Propose(context.Background(), p, "atlas", nil, installed)
	if err != nil {
		t.Fatal(err)
	}
	if prop.Cue.Name != "Atlas" {
		t.Errorf("name = %q", prop.Cue.Name)
	}
	rf := body["response_format"].(map[string]any)
	if rf["type"] != "json_schema" {
		t.Errorf("response_format = %v", rf)
	}

	missing, _ := New(Config{Provider: ProviderOpenAI, BaseURL: srv.URL + "/v1", Model: "nope"}, "")
	if err := missing.Check(context.Background()); err == nil || !strings.Contains(err.Error(), "nope") {
		t.Errorf("Check with unknown model = %v", err)
	}
}

func TestAnthropicRequestShape(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "test-key" {
			w.WriteHeader(401)
			io.WriteString(w, `{"type":"error","error":{"type":"authentication_error","message":"bad key"}}`)
			return
		}
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": "msg_1", "type": "message", "role": "assistant", "model": "claude-opus-5-5",
			"stop_reason": "end_turn",
			"content":     []any{map[string]any{"type": "text", "text": atlasAnswer}},
			"usage":       map[string]any{"input_tokens": 1, "output_tokens": 1},
		})
	}))
	defer srv.Close()

	p := NewAnthropic("test-key", DefaultAnthropicModel, srv.URL)
	prop, err := Propose(context.Background(), p, "atlas", nil, installed)
	if err != nil {
		t.Fatal(err)
	}
	if prop.Cue.Name != "Atlas" {
		t.Errorf("name = %q", prop.Cue.Name)
	}
	if body["model"] != "claude-opus-5-5" {
		t.Errorf("model = %v", body["model"])
	}
	oc := body["output_config"].(map[string]any)
	if oc["format"].(map[string]any)["type"] != "json_schema" {
		t.Errorf("output_config = %v", oc)
	}

	bad := NewAnthropic("wrong", DefaultAnthropicModel, srv.URL)
	if _, err := bad.Complete(context.Background(), Request{System: "s", User: "u", Schema: draftSchema}); err == nil || !strings.Contains(err.Error(), "API key") {
		t.Errorf("bad key error = %v", err)
	}
}

func TestNewValidatesConfig(t *testing.T) {
	cases := []Config{
		{},
		{Provider: ProviderAnthropic},
		{Provider: ProviderOpenAI, Model: "x"},
		{Provider: ProviderOpenAI, BaseURL: "http://localhost:11434/v1"},
		{Provider: "other"},
	}
	for _, c := range cases {
		if _, err := New(c, ""); err == nil {
			t.Errorf("New(%+v) should fail", c)
		}
	}
}

func TestOpenAICompatConnectionRefused(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close() // nothing listens here any more
	p := NewOpenAICompat(url+"/v1", "", "m")
	err := p.Check(context.Background())
	if err == nil || !strings.Contains(err.Error(), "Is the server running?") {
		t.Errorf("got %v", err)
	}
}

func TestDescribe(t *testing.T) {
	cases := []struct {
		cfg  Config
		want Disclosure
	}{
		{Config{Provider: ProviderAnthropic}, Disclosure{Model: "Claude Opus 5.5", Vendor: "Anthropic"}},
		{Config{Provider: ProviderAnthropic, Model: "claude-haiku-5-5"}, Disclosure{Model: "Claude Haiku 5.5", Vendor: "Anthropic"}},
		{Config{Provider: ProviderOpenAI, Model: "llama3.2", BaseURL: "http://localhost:11434/v1"}, Disclosure{Model: "llama3.2", Vendor: "Ollama", Local: true}},
		{Config{Provider: ProviderOpenAI, Model: "qwen3", BaseURL: "http://127.0.0.1:1234/v1"}, Disclosure{Model: "qwen3", Vendor: "LM Studio", Local: true}},
		{Config{Provider: ProviderOpenAI, Model: "gpt-5", BaseURL: "https://api.openai.com/v1"}, Disclosure{Model: "gpt-5", Vendor: "OpenAI"}},
		{Config{Provider: ProviderOpenAI, Model: "x", BaseURL: "https://llm.example.com/v1"}, Disclosure{Model: "x", Vendor: "llm.example.com"}},
		{Config{}, Disclosure{}},
	}
	for _, c := range cases {
		if got := Describe(c.cfg); got != c.want {
			t.Errorf("Describe(%+v) = %+v, want %+v", c.cfg, got, c.want)
		}
	}
}
