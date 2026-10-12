package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// The exact error Gemini returned to a user.
const geminiBusy = `[{ "error": { "code": 503, "message": "This model is currently experiencing high demand. Spikes in demand are usually temporary. Please try again later.", "status": "UNAVAILABLE" } } ]`

func fastRetries(t *testing.T) {
	old := retryDelays
	retryDelays = []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}
	t.Cleanup(func() { retryDelays = old })
}

// geminiLike answers 503 for the busy models and succeeds for the others,
// recording which models were asked.
func geminiLike(t *testing.T, busy map[string]int) (*httptest.Server, *[]string) {
	var mu sync.Mutex
	var asked []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		model, _ := body["model"].(string)
		mu.Lock()
		asked = append(asked, model)
		left, isBusy := busy[model]
		if isBusy && left != 0 {
			busy[model] = left - 1 // -1 means busy forever
			mu.Unlock()
			w.WriteHeader(503)
			io.WriteString(w, geminiBusy)
			return
		}
		mu.Unlock()
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{
			"message": map[string]any{"role": "assistant", "content": atlasAnswer}, "finish_reason": "stop",
		}}})
	}))
	t.Cleanup(srv.Close)
	return srv, &asked
}

func TestBusyIsRetriedQuietly(t *testing.T) {
	fastRetries(t)
	srv, asked := geminiLike(t, map[string]int{"gemini-3.5-flash": 2}) // busy twice, then fine
	p := NewOpenAICompat(srv.URL, "k", "gemini-3.5-flash")
	var statuses []string
	ctx := WithStatus(context.Background(), func(s string) { statuses = append(statuses, s) })
	prop, err := Propose(ctx, p, "atlas", nil, installed)
	if err != nil {
		t.Fatal(err)
	}
	if len(*asked) != 3 || prop.Model != "" {
		t.Errorf("asked %q, model note %q", *asked, prop.Model)
	}
	if len(statuses) != 2 || !strings.Contains(statuses[0], "busy, trying again") {
		t.Errorf("statuses = %q", statuses)
	}
}

func TestBackupModelStepsIn(t *testing.T) {
	fastRetries(t)
	srv, asked := geminiLike(t, map[string]int{"gemini-3.5-flash": -1})
	p := NewOpenAICompat(srv.URL, "k", "gemini-3.5-flash")
	p.fallbacks = []string{"gemini-2.5-flash", "gemini-2.5-flash-lite"}
	var statuses []string
	ctx := WithStatus(context.Background(), func(s string) { statuses = append(statuses, s) })
	prop, err := Propose(ctx, p, "atlas", nil, installed)
	if err != nil {
		t.Fatal(err)
	}
	if prop.Model != "Gemini 2.5 Flash" {
		t.Errorf("proposal should say which model answered, got %q", prop.Model)
	}
	if last := (*asked)[len(*asked)-1]; last != "gemini-2.5-flash" {
		t.Errorf("asked %q", *asked)
	}
	if !strings.Contains(strings.Join(statuses, "|"), "Gemini 3.5 Flash is busy, trying Gemini 2.5 Flash") {
		t.Errorf("statuses = %q", statuses)
	}
}

func TestStillBusyGivesAPlainMessage(t *testing.T) {
	fastRetries(t)
	srv, _ := geminiLike(t, map[string]int{"gemini-3.5-flash": -1, "gemini-2.5-flash": -1})
	p := NewOpenAICompat(srv.URL, "k", "gemini-3.5-flash")
	p.vendor = "Gemini"
	p.fallbacks = []string{"gemini-2.5-flash"}
	_, err := p.Complete(context.Background(), Request{System: "s", User: "u", Schema: draftSchema})
	var e *Error
	if !errors.As(err, &e) || e.Kind != KindBusy || !e.TriedBackup {
		t.Fatalf("err = %#v", err)
	}
	msg := err.Error()
	for _, bad := range []string{"http", "503", "{", "UNAVAILABLE"} {
		if strings.Contains(msg, bad) {
			t.Errorf("message leaks %q: %s", bad, msg)
		}
	}
	if !strings.HasPrefix(msg, "Gemini is very busy right now.") {
		t.Errorf("message = %s", msg)
	}
}

func TestClassifyHTTP(t *testing.T) {
	cases := []struct {
		status int
		body   string
		kind   ErrorKind
		after  time.Duration
	}{
		{503, geminiBusy, KindBusy, 0},
		{529, "overloaded", KindBusy, 0},
		{429, `{"error":{"message":"Quota exceeded for metric: generate_content_free_tier_requests, limit: 10, quotaId: GenerateRequestsPerMinutePerProjectPerModel-FreeTier","details":[{"retryDelay":"17s"}]}}`, KindRateLimit, 17 * time.Second},
		{429, `{"error":{"message":"quotaId: GenerateRequestsPerDayPerProjectPerModel-FreeTier"}}`, KindQuota, 0},
		{400, `{"error":{"message":"API key not valid. Please pass a valid API key."}}`, KindAuth, 0},
		{401, ``, KindAuth, 0},
		{404, `model not found`, KindModel, 0},
		{400, `unknown field additionalProperties`, KindBadRequest, 0},
	}
	for _, c := range cases {
		e := classifyHTTP(c.status, c.body, "Gemini", "m")
		if e.Kind != c.kind || e.RetryAfter != c.after {
			t.Errorf("%d %q → %s (%v), want %s (%v)", c.status, c.body[:min(30, len(c.body))], e.Kind, e.RetryAfter, c.kind, c.after)
		}
	}
}
