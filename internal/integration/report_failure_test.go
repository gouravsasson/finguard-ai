package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"finguard.ai/internal/aigateway"
	"finguard.ai/internal/domain"
	"finguard.ai/internal/report"
	"finguard.ai/internal/seed"
)

type injectedProvider func(context.Context, aigateway.Request) (aigateway.Response, error)

func (p injectedProvider) Generate(ctx context.Context, req aigateway.Request) (aigateway.Response, error) {
	return p(ctx, req)
}

func TestReportFailuresPreserveCase(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	if err := seed.Run(ctx, e.store, e.admin.Pool, "pw", true); err != nil {
		t.Fatal(err)
	}
	e.login("senior@northwind.demo", "pw")
	row := e.get("/api/v1/cases?limit=1")["data"].([]any)[0].(map[string]any)
	id := row["id"].(string)
	p := domain.Principal{TenantID: seed.TenantNorthwind, Role: domain.RoleSenior, Email: "system"}
	for _, scenario := range []struct {
		name, status string
		provider     injectedProvider
	}{
		{"timeout", "failed", func(ctx context.Context, _ aigateway.Request) (aigateway.Response, error) {
			<-ctx.Done()
			return aigateway.Response{}, ctx.Err()
		}},
		{"rate-limit", "failed", func(context.Context, aigateway.Request) (aigateway.Response, error) {
			return aigateway.Response{}, errors.New("429 rate limit")
		}},
		{"invalid-json", "failed", func(context.Context, aigateway.Request) (aigateway.Response, error) {
			return aigateway.Response{Text: "not json"}, nil
		}},
		{"tool-request", "rejected", func(context.Context, aigateway.Request) (aigateway.Response, error) {
			return aigateway.Response{Text: `{"summary":"x","tool_call":"freeze_account"}`}, nil
		}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			svc := &report.Service{Store: e.store, Provider: scenario.provider, ProviderName: "injected-test-provider", Timeout: time.Millisecond, MaxChars: 24000, MaxTokens: 800}
			result, err := svc.Generate(ctx, p, id)
			if err == nil || result.Status != scenario.status || result.Live {
				t.Fatalf("status=%s live=%v err=%v", result.Status, result.Live, err)
			}
			current := e.get("/api/v1/cases/" + id)["case"].(map[string]any)
			if current["status"] != row["status"] || current["version"] != row["version"] {
				t.Fatal("failed model call changed the case")
			}
			if len(e.get("/api/v1/cases/" + id + "/evidence")["data"].([]any)) == 0 {
				t.Fatal("evidence unavailable after failure")
			}
		})
	}
}
