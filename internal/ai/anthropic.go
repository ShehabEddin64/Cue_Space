package ai

import (
	"context"
	"errors"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/shared/constant"

	"github.com/ShehabEddin64/Cue_Space/internal/i18n"
)

type Anthropic struct {
	client  anthropic.Client
	model   string
	baseURL string
}

func NewAnthropic(apiKey, model, baseURL string) *Anthropic {
	// The SDK retries overloads and rate limits itself; allow a few more.
	opts := []option.RequestOption{option.WithAPIKey(apiKey), option.WithMaxRetries(4)}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	return &Anthropic{client: anthropic.NewClient(opts...), model: model, baseURL: baseURL}
}

func (a *Anthropic) Complete(ctx context.Context, req Request) (string, error) {
	params := anthropic.BetaMessageNewParams{
		Model:     a.model,
		MaxTokens: 16000,
		System:    []anthropic.BetaTextBlockParam{{Text: req.System}},
		Messages: []anthropic.BetaMessageParam{
			anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(req.User)),
		},
		OutputConfig: anthropic.BetaOutputConfigParam{
			Effort: anthropic.BetaOutputConfigEffortMedium,
			Format: anthropic.BetaJSONOutputFormatParam{Schema: req.Schema},
		},
	}
	if a.baseURL == "" {
		// On the Claude API, let a safety refusal be re-served by a suitable
		// fallback model instead of failing the proposal.
		params.Betas = []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01}
		params.Fallbacks = anthropic.BetaFallbacksParamUnion{OfDefault: constant.ValueOf[constant.Default]()}
	}
	resp, err := a.client.Beta.Messages.New(ctx, params)
	if err != nil {
		return "", describeAnthropicError(err)
	}
	switch resp.StopReason {
	case anthropic.BetaStopReasonRefusal:
		return "", &Error{Kind: KindRefused, Vendor: "Claude", Model: a.model}
	case anthropic.BetaStopReasonMaxTokens:
		return "", i18n.Errorf("The answer was cut off. Try a shorter description.")
	}
	var out strings.Builder
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.BetaTextBlock); ok {
			out.WriteString(t.Text)
		}
	}
	return out.String(), nil
}

// Check looks up the configured model, which validates the key without
// generating any tokens.
func (a *Anthropic) Check(ctx context.Context) error {
	if _, err := a.client.Models.Get(ctx, a.model, anthropic.ModelGetParams{}); err != nil {
		e := describeAnthropicError(err)
		var ae *Error
		if errors.As(e, &ae) && ae.Kind == KindModel {
			ae.Model = a.model
		}
		return e
	}
	return nil
}

func (a *Anthropic) Models(ctx context.Context) ([]string, error) {
	var ids []string
	pager := a.client.Models.ListAutoPaging(ctx, anthropic.ModelListParams{})
	for pager.Next() {
		ids = append(ids, pager.Current().ID)
	}
	if err := pager.Err(); err != nil {
		return nil, describeAnthropicError(err)
	}
	return ids, nil
}

func describeAnthropicError(err error) error {
	if errors.Is(err, context.Canceled) {
		return err
	}
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) {
		e := classifyHTTP(apiErr.StatusCode, apiErr.Error(), "Claude", "")
		if apiErr.StatusCode == 529 {
			e.Kind = KindBusy // "overloaded"
		}
		return e
	}
	return &Error{Kind: KindNetwork, Vendor: "Claude", Detail: err.Error()}
}
