package ai

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ErrorKind says what went wrong in terms a person can act on.
type ErrorKind string

const (
	KindBusy       ErrorKind = "busy"       // overloaded or briefly down (5xx): retry
	KindRateLimit  ErrorKind = "rate-limit" // too many requests per minute: wait, retry
	KindQuota      ErrorKind = "quota"      // the free allowance for today is used up
	KindAuth       ErrorKind = "auth"       // the API key was rejected
	KindModel      ErrorKind = "model"      // the model doesn't exist for this key
	KindNetwork    ErrorKind = "network"    // the server couldn't be reached
	KindLocalOff   ErrorKind = "local-off"  // a local server (Ollama, LM Studio) isn't running
	KindRefused    ErrorKind = "refused"    // the model declined the request
	KindBadRequest ErrorKind = "bad-request"
	KindOther      ErrorKind = "other"
)

// Error is an AI failure with a message written for people. The raw server
// response is kept in Detail for logs, never shown as the message.
type Error struct {
	Kind   ErrorKind
	Vendor string
	Model  string
	Detail string
	// RetryAfter is the server's own hint, when it gave one.
	RetryAfter time.Duration
	// TriedBackup is true when backup models were tried as well.
	TriedBackup bool
}

func (e *Error) Error() string {
	v := e.Vendor
	if v == "" {
		v = "The AI service"
	}
	switch e.Kind {
	case KindBusy:
		also := ""
		if e.TriedBackup {
			also = " and a backup model"
		}
		return fmt.Sprintf("%s is very busy right now. Cue Space tried again%s, but it’s still overloaded. Please try again in a minute.", v, also)
	case KindRateLimit:
		return fmt.Sprintf("You’re sending requests faster than %s allows on your plan. Wait a minute, then try again.", v)
	case KindQuota:
		return fmt.Sprintf("You’ve used up today’s free %s requests. Try again tomorrow, or choose another model or AI in Settings.", v)
	case KindAuth:
		return fmt.Sprintf("%s didn’t accept your API key. Check it in Settings.", v)
	case KindModel:
		if e.Model != "" {
			return fmt.Sprintf("The model “%s” isn’t available with your key. Choose another one in Settings.", e.Model)
		}
		return "That model isn’t available with your key. Choose another one in Settings."
	case KindNetwork:
		return fmt.Sprintf("Cue Space can’t reach %s. Check your internet connection and try again.", v)
	case KindLocalOff:
		return fmt.Sprintf("%s isn’t running. Open it, then try again.", v)
	case KindRefused:
		return fmt.Sprintf("%s declined this request. Try describing it differently.", v)
	}
	return fmt.Sprintf("%s couldn’t answer this time. Please try again.", v)
}

// Temporary errors are worth retrying automatically.
func (e *Error) Temporary() bool {
	return e.Kind == KindBusy || e.Kind == KindRateLimit || e.Kind == KindNetwork
}

// classifyHTTP turns an HTTP failure into an Error.
func classifyHTTP(status int, body, vendor, model string) *Error {
	e := &Error{Vendor: vendor, Model: model, Detail: fmt.Sprintf("HTTP %d: %s", status, body)}
	lower := strings.ToLower(body)
	switch {
	case status == 401 || status == 403:
		e.Kind = KindAuth
	case status == 404:
		e.Kind = KindModel
	case status == 429:
		// Free tiers have per-minute and per-day limits; only the first passes.
		if strings.Contains(body, "PerDay") || strings.Contains(lower, "per day") || strings.Contains(lower, "daily") {
			e.Kind = KindQuota
		} else {
			e.Kind = KindRateLimit
			e.RetryAfter = retryDelay(body)
		}
	case status == 408 || status >= 500:
		e.Kind = KindBusy
	case status == 400 || status == 422:
		e.Kind = KindBadRequest
		if strings.Contains(lower, "api key") || strings.Contains(lower, "api_key") {
			e.Kind = KindAuth
		}
	default:
		e.Kind = KindOther
	}
	return e
}

var retryDelayRe = regexp.MustCompile(`"retryDelay"\s*:\s*"(\d+(?:\.\d+)?)s"`)

// retryDelay reads Google's RetryInfo hint ("retryDelay": "17s").
func retryDelay(body string) time.Duration {
	m := retryDelayRe.FindStringSubmatch(body)
	if m == nil {
		return 0
	}
	secs, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0
	}
	return time.Duration(secs * float64(time.Second))
}

// ---- status updates ---------------------------------------------------------

type statusKey struct{}

// WithStatus attaches a callback that receives short progress messages, such
// as "Gemini is busy, trying again…", while a request is retried.
func WithStatus(ctx context.Context, report func(string)) context.Context {
	return context.WithValue(ctx, statusKey{}, report)
}

func reportStatus(ctx context.Context, msg string) {
	if f, ok := ctx.Value(statusKey{}).(func(string)); ok && f != nil {
		f(msg)
	}
}

// ---- retrying -----------------------------------------------------------------

// retryDelays are the waits between attempts for temporary failures.
var retryDelays = []time.Duration{1 * time.Second, 2500 * time.Millisecond, 5 * time.Second}

// maxWait caps how long Cue Space waits on a server's own retry hint before
// giving up and saying so.
const maxWait = 25 * time.Second

// withRetry runs fn, retrying temporary failures with growing waits.
// retries is how many extra attempts are allowed.
func withRetry(ctx context.Context, retries int, fn func() (string, error)) (string, error) {
	for attempt := 0; ; attempt++ {
		out, err := fn()
		var e *Error
		if err == nil || !errors.As(err, &e) || !e.Temporary() || attempt >= retries || attempt >= len(retryDelays) {
			return out, err
		}
		wait := retryDelays[attempt]
		if e.RetryAfter > 0 {
			if e.RetryAfter > maxWait {
				return out, err
			}
			wait = e.RetryAfter + 500*time.Millisecond
		}
		reportStatus(ctx, retryStatus(e, wait))
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(wait):
		}
	}
}

func retryStatus(e *Error, wait time.Duration) string {
	v := e.Vendor
	if v == "" {
		v = "The AI service"
	}
	switch e.Kind {
	case KindRateLimit:
		return fmt.Sprintf("Waiting %ds for %s’s rate limit", int(wait.Round(time.Second)/time.Second), v)
	case KindNetwork:
		return fmt.Sprintf("Can’t reach %s, trying again", v)
	}
	return fmt.Sprintf("%s is busy, trying again", v)
}

// completeWithBackups asks models in order: the chosen one first, with a
// couple of quick retries, then each backup once. Busy, rate-limited,
// out-of-quota or missing models move on to the next; anything else stops.
// It returns the answer and the model that gave it.
func completeWithBackups(ctx context.Context, models []string, ask func(model string) (string, error)) (string, string, error) {
	var lastErr error
	for i, model := range models {
		if i > 0 {
			reportStatus(ctx, prettyModel(models[i-1])+" is busy, trying "+prettyModel(model))
		}
		// With backups available, don't keep a person waiting on a busy
		// model: two quick retries, then switch.
		retries := len(retryDelays)
		if len(models) > 1 {
			retries = 2
		}
		if i > 0 {
			retries = 1
		}
		out, err := withRetry(ctx, retries, func() (string, error) { return ask(model) })
		if err == nil {
			return out, model, nil
		}
		lastErr = err
		var e *Error
		if !errors.As(err, &e) {
			return "", "", err
		}
		switch e.Kind {
		case KindBusy, KindRateLimit, KindQuota, KindModel:
			continue // a backup model may well work
		}
		return "", "", err
	}
	var e *Error
	if errors.As(lastErr, &e) && len(models) > 1 {
		e.TriedBackup = true
		e.Model = models[0]
	}
	return "", "", lastErr
}
