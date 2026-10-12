package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/shared/constant"
)

type Anthropic struct {
	client  anthropic.Client
	model   string
	baseURL string
}

func NewAnthropic(apiKey, model, baseURL string) *Anthropic {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
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
		return "", errors.New("the model declined this request")
	case anthropic.BetaStopReasonMaxTokens:
		return "", errors.New("the model's answer was cut off; try a shorter description")
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
		return describeAnthropicError(err)
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
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 401:
			return errors.New("Anthropic rejected the API key")
		case 403:
			return errors.New("this API key cannot use that model")
		case 404:
			return errors.New("model not found; check the model name in Settings")
		case 429:
			return errors.New("rate limited by Anthropic; try again in a moment")
		case 529:
			return errors.New("Anthropic is overloaded; try again in a moment")
		}
		if apiErr.StatusCode >= 500 {
			return fmt.Errorf("Anthropic server error (%d); try again", apiErr.StatusCode)
		}
		return fmt.Errorf("Anthropic request failed (%d)", apiErr.StatusCode)
	}
	if errors.Is(err, context.Canceled) {
		return err
	}
	return fmt.Errorf("could not reach Anthropic: %w", err)
}
