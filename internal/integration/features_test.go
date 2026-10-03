package integration

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"finguard.ai/internal/seed"
)

func TestKnockoutDesk(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	if err := seed.Run(ctx, e.store, e.admin.Pool, "pw", true); err != nil {
		t.Fatal(err)
	}
	e.login("investigator@northwind.demo", "pw")
	page := e.get("/api/v1/cases?limit=50")
	var caseID string
	for _, row := range page["data"].([]any) {
		item := row.(map[string]any)
		if item["account_external_id"] == "acct_milo" {
			caseID = item["id"].(string)
		}
	}
	if caseID == "" {
		t.Fatal("milo case missing")
	}
	board := e.get("/api/v1/cases/" + caseID + "/knockouts")
	if board["method"] != "deterministic-replay" {
		t.Fatalf("method %#v", board["method"])
	}
	var stayed, cleared bool
	for _, raw := range board["signals"].([]any) {
		sig := raw.(map[string]any)
		if sig["rule_code"] != "velocity" || sig["reproduced"] != true {
			continue
		}
		for _, item := range sig["attempts"].([]any) {
			attempt := item.(map[string]any)
			if attempt["clears"].(bool) {
				cleared = true
			} else {
				stayed = true
			}
			if attempt["detail"] == "" {
				t.Fatal("blank knockout detail")
			}
		}
	}
	if !stayed || !cleared {
		t.Fatalf("expected a replay that stays and one that clears: %#v", board["signals"])
	}
	dissent := e.post("/api/v1/cases/"+caseID+"/dissent", map[string]any{})
	if dissent["live"] != false || dissent["provider"] != "local-stub" {
		t.Fatalf("dissent labeled incorrectly %#v", dissent)
	}
	if dissent["concern"].(map[string]any)["claim"] == "" {
		t.Fatal("empty concern")
	}
	e.login("viewer@northwind.demo", "pw")
	e.get("/api/v1/cases/" + caseID + "/knockouts")
	e.postStatus("/api/v1/cases/"+caseID+"/dissent", map[string]any{}, http.StatusForbidden)
}

func TestSupportingEndpoints(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	if err := seed.Run(ctx, e.store, e.admin.Pool, "pw", true); err != nil {
		t.Fatal(err)
	}
	e.getStatus("/api/v1/dashboard", http.StatusUnauthorized)
	e.login("admin@northwind.demo", "pw")
	for _, path := range []string{"/healthz", "/readyz", "/api/v1/auth/me", "/api/v1/meta", "/api/v1/dashboard", "/api/v1/metrics", "/api/v1/transactions?limit=2"} {
		e.get(path)
	}
	first := e.get("/api/v1/cases?limit=2")
	rows := first["data"].([]any)
	if len(rows) != 2 {
		t.Fatal("pagination limit not respected")
	}
	cursor := first["page"].(map[string]any)["next_cursor"].(string)
	second := e.get("/api/v1/cases?limit=2&cursor=" + url.QueryEscape(cursor))
	for _, a := range rows {
		for _, b := range second["data"].([]any) {
			if a.(map[string]any)["id"] == b.(map[string]any)["id"] {
				t.Fatal("duplicate case across pages")
			}
		}
	}
	e.getStatus("/api/v1/cases?cursor=invalid", http.StatusUnprocessableEntity)
	caseID := rows[0].(map[string]any)["id"].(string)
	for _, suffix := range []string{"", "/evidence", "/reports", "/graph", "/knockouts", "/audit", "/assistant"} {
		e.get("/api/v1/cases/" + caseID + suffix)
	}
	e.post("/api/v1/cases/"+caseID+"/assistant", map[string]any{"question": "What evidence is missing?"})
	if _, err := e.admin.Pool.Exec(ctx, `UPDATE assistant_messages SET retain_until = now() - interval '1 day' WHERE tenant_id = $1`, seed.TenantNorthwind); err != nil {
		t.Fatal(err)
	}
	if e.post("/api/v1/admin/retention", map[string]any{})["deleted"].(float64) == 0 {
		t.Fatal("expired assistant messages were not purged")
	}
	var jobID string
	if err := e.admin.Pool.QueryRow(ctx, `SELECT id FROM jobs WHERE tenant_id=$1 LIMIT 1`, seed.TenantNorthwind).Scan(&jobID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.admin.Pool.Exec(ctx, `UPDATE jobs SET status='dead', attempts=3 WHERE id=$1`, jobID); err != nil {
		t.Fatal(err)
	}
	if e.post("/api/v1/jobs/"+jobID+"/replay", map[string]any{})["status"] != "pending" {
		t.Fatal("job was not replayed")
	}
	e.postStatus("/api/v1/jobs/"+jobID+"/replay", map[string]any{}, http.StatusNotFound)
	payload := event("batch-check", "acct_batch_check", time.Now().UTC(), 1000, "approved", "synthetic")
	result := e.post("/api/v1/events/batch", map[string]any{"events": []any{payload, payload}})["results"].([]any)
	if result[0].(map[string]any)["status"] != "accepted" || result[1].(map[string]any)["status"] != "replayed" {
		t.Fatalf("batch: %#v", result)
	}
	e.postStatus("/api/v1/events/batch", map[string]any{"events": []any{}}, http.StatusUnprocessableEntity)
	e.login("investigator@southbank.demo", "pw")
	for _, suffix := range []string{"", "/evidence", "/reports", "/graph", "/knockouts", "/audit", "/assistant"} {
		e.getStatus("/api/v1/cases/"+caseID+suffix, http.StatusNotFound)
	}
	e.postStatus("/api/v1/cases/"+caseID+"/notes", map[string]any{"body": "Must not cross tenant boundaries"}, http.StatusNotFound)
	e.postStatus("/api/v1/admin/retention", map[string]any{}, http.StatusForbidden)
	e.post("/api/v1/auth/logout", map[string]any{})
	e.getStatus("/api/v1/auth/me", http.StatusUnauthorized)
}
