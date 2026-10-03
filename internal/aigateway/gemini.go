package aigateway

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"google.golang.org/genai"
)

// Gemini is the live adapter. It never attaches tools.
type Gemini struct {
	APIKey string
	Model  string
	HTTP   *http.Client

	once    sync.Once
	client  *genai.Client
	initErr error
}

func (g *Gemini) Generate(ctx context.Context, req Request) (Response, error) {
	if g.APIKey == "" {
		return Response{}, errors.New("GEMINI_API_KEY is not configured")
	}
	g.once.Do(func() {
		httpClient := g.HTTP
		if httpClient == nil {
			httpClient = &http.Client{Timeout: 25 * time.Second}
		}
		g.client, g.initErr = genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:     g.APIKey,
			Backend:    genai.BackendGeminiAPI,
			HTTPClient: httpClient,
		})
	})
	if g.initErr != nil {
		return Response{}, g.initErr
	}
	model := g.Model
	if model == "" {
		model = "gemini-2.5-flash"
	}
	maxOut := req.MaxOutputTokens
	if maxOut <= 0 {
		maxOut = 1500
	}
	cfg := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{Parts: []*genai.Part{{Text: req.System}}},
		Temperature:       genai.Ptr(float32(0.2)),
		MaxOutputTokens:   int32(maxOut),
		ResponseMIMEType:  "application/json",
		Seed:              genai.Ptr(int32(7)),
		// Tools intentionally omitted. The model must not be given a way to act.
	}
	if cfg.Tools != nil {
		return Response{}, errors.New("gemini config must not include tools")
	}
	start := time.Now()
	resp, err := g.client.Models.GenerateContent(ctx, model, []*genai.Content{
		{Role: "user", Parts: []*genai.Part{{Text: req.User}}},
	}, cfg)
	if err != nil {
		return Response{}, err
	}
	if resp == nil {
		return Response{}, errors.New("empty gemini response")
	}
	if len(resp.FunctionCalls()) > 0 {
		return Response{}, fmt.Errorf("%w: model requested a function call", ErrPoison)
	}
	out := Response{
		Text:     resp.Text(),
		Model:    model,
		Provider: "gemini",
		Live:     true,
		Latency:  time.Since(start),
	}
	if resp.ModelVersion != "" {
		out.Model = resp.ModelVersion
	}
	if resp.UsageMetadata != nil {
		out.InputTokens = int(resp.UsageMetadata.PromptTokenCount)
		out.OutputTokens = int(resp.UsageMetadata.CandidatesTokenCount)
	}
	if out.Text == "" {
		return Response{}, errors.New("gemini returned no text")
	}
	return out, nil
}
