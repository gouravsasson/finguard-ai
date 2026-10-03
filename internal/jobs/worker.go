package jobs

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"finguard.ai/internal/aigateway"
	"finguard.ai/internal/domain"
	"finguard.ai/internal/report"
	"finguard.ai/internal/store"
)

type Worker struct {
	Store  *store.Store
	Report *report.Service
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.Drain(ctx)
		}
	}
}

func (w *Worker) Drain(ctx context.Context) {
	for {
		ok, err := w.ProcessOnce(ctx)
		if err != nil {
			slog.Error("job processing failed", "error", err.Error())
			return
		}
		if !ok {
			return
		}
	}
}

func (w *Worker) ProcessOnce(ctx context.Context) (bool, error) {
	job, ok, err := w.Store.ClaimJob(ctx)
	if err != nil || !ok {
		return false, err
	}
	caseID, _ := job.Payload["case_id"].(string)
	_, genErr := w.Report.Generate(ctx, domain.Principal{TenantID: job.TenantID, Role: domain.RoleAdmin, Email: "system"}, caseID)
	if genErr == nil || errors.Is(genErr, aigateway.ErrPoison) {
		return true, w.Store.FinishJob(ctx, job.ID, "done", "", job.Attempts, time.Now().UTC())
	}
	attempts := job.Attempts + 1
	status := "pending"
	if attempts >= job.MaxAttempts {
		status = "dead"
	}
	next := time.Now().UTC().Add(Backoff(attempts))
	if err := w.Store.FinishJob(ctx, job.ID, status, clip(genErr.Error(), 300), attempts, next); err != nil {
		return false, err
	}
	return true, nil
}

func Backoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	base := time.Second << (attempt - 1)
	if base > 30*time.Second {
		base = 30 * time.Second
	}
	return base
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
