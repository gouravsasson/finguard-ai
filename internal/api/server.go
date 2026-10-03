package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"finguard.ai/internal/aigateway"
	"finguard.ai/internal/auth"
	"finguard.ai/internal/casesvc"
	"finguard.ai/internal/config"
	"finguard.ai/internal/domain"
	"finguard.ai/internal/graph"
	"finguard.ai/internal/httpx"
	"finguard.ai/internal/idgen"
	"finguard.ai/internal/ingestion"
	"finguard.ai/internal/knockout"
	"finguard.ai/internal/metrics"
	"finguard.ai/internal/report"
	"finguard.ai/internal/store"
	"finguard.ai/internal/web"
)

type Server struct {
	Config    config.Config
	Store     *store.Store
	Auth      *auth.Service
	Ingest    *ingestion.Service
	Cases     *casesvc.Service
	Reports   *report.Service
	Graph     *graph.Service
	Knockouts *knockout.Service
	Metrics   *metrics.Metrics
	Log       *slog.Logger
	loginCap  *httpx.Limiter
}

func (s *Server) Handler() http.Handler {
	if s.Log == nil {
		s.Log = slog.Default()
	}
	if s.Metrics == nil {
		s.Metrics = &metrics.Metrics{}
	}
	s.loginCap = httpx.NewLimiter(s.Config.LoginLimitPerMin)
	mux := http.NewServeMux()
	for _, rt := range s.routeTable() {
		mux.HandleFunc(rt.Method+" "+rt.Path, rt.Handler)
	}
	mux.HandleFunc("/", s.static)
	return httpx.Middleware(s.Log, s.Config.BodyLimit, s.Config.CookieSecure, s.Config.CORSOrigin, httpx.NewLimiter(s.Config.RateLimitPerMin), mux)
}

type Route struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
}

func (s *Server) routeTable() []Route {
	return []Route{
		{"GET", "/healthz", s.health},
		{"GET", "/readyz", s.ready},
		{"GET", "/api/v1/openapi.yaml", s.openAPI},
		{"POST", "/api/v1/auth/login", s.login},
		{"POST", "/api/v1/auth/logout", s.logout},
		{"GET", "/api/v1/auth/me", s.me},
		{"GET", "/api/v1/meta", s.meta},
		{"GET", "/api/v1/dashboard", s.dashboard},
		{"POST", "/api/v1/events", s.ingestOne},
		{"POST", "/api/v1/events/batch", s.ingestBatch},
		{"GET", "/api/v1/transactions", s.transactions},
		{"GET", "/api/v1/cases", s.listCases},
		{"GET", "/api/v1/cases/{id}", s.getCase},
		{"POST", "/api/v1/cases/{id}/notes", s.addNote},
		{"POST", "/api/v1/cases/{id}/actions", s.act},
		{"GET", "/api/v1/cases/{id}/evidence", s.evidence},
		{"GET", "/api/v1/cases/{id}/reports", s.listReports},
		{"POST", "/api/v1/cases/{id}/reports", s.createReport},
		{"GET", "/api/v1/cases/{id}/graph", s.caseGraph},
		{"GET", "/api/v1/cases/{id}/knockouts", s.knockouts},
		{"POST", "/api/v1/cases/{id}/dissent", s.dissent},
		{"GET", "/api/v1/cases/{id}/audit", s.audit},
		{"GET", "/api/v1/cases/{id}/assistant", s.listAssistant},
		{"POST", "/api/v1/cases/{id}/assistant", s.ask},
		{"POST", "/api/v1/jobs/{id}/replay", s.replayJob},
		{"POST", "/api/v1/admin/retention", s.retention},
		{"GET", "/api/v1/metrics", s.metrics},
	}
}

func (s *Server) Patterns() []string {
	var out []string
	for _, rt := range s.routeTable() {
		out = append(out, rt.Method+" "+rt.Path)
	}
	return out
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.Store.Pool.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "not_ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ready"})
}

func (s *Server) openAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	_, _ = w.Write(openAPIYAML)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if !csrf(r) {
		s.fail(w, r, http.StatusForbidden, "csrf", "Missing X-FinGuard-Request header")
		return
	}
	if s.loginCap != nil && !s.loginCap.Allow(r.RemoteAddr) {
		s.fail(w, r, http.StatusTooManyRequests, "rate_limited", "Too many login attempts")
		return
	}
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decode(r, &body); err != nil {
		s.fail(w, r, http.StatusBadRequest, "invalid_json", "Request body must be JSON")
		return
	}
	token, p, err := s.Auth.Login(r.Context(), strings.TrimSpace(body.Email), body.Password)
	if err != nil {
		s.writeDomain(w, r, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: "finguard_session", Value: token, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: s.Config.CookieSecure, MaxAge: int(s.Config.SessionTTL.Seconds()),
	})
	writeJSON(w, http.StatusOK, map[string]any{"user": publicUser(p)})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if !csrf(r) {
		s.fail(w, r, http.StatusForbidden, "csrf", "Missing X-FinGuard-Request header")
		return
	}
	if raw := tokenFrom(r); raw != "" {
		_ = s.Auth.Logout(r.Context(), raw)
	}
	http.SetCookie(w, &http.Cookie{Name: "finguard_session", Value: "", Path: "/", HttpOnly: true, MaxAge: -1})
	writeJSON(w, http.StatusOK, map[string]any{"status": "logged_out"})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	p, ok := s.require(w, r, "read")
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": publicUser(p)})
}

func (s *Server) meta(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.require(w, r, "read"); !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.metaBody())
}

func (s *Server) metaBody() map[string]any {
	return map[string]any{
		"product":         "FinGuard AI",
		"synthetic_only":  true,
		"adverse_actions": false,
		"ai_provider":     s.Reports.ProviderName,
		"ai_live":         s.Config.LiveModel() && s.Reports.ProviderName == "gemini",
		"model":           s.Config.GeminiModel,
		"prompt_version":  aigateway.ReportPromptVersion,
		"limitations": []string{
			"Synthetic demonstration data only.",
			"The assistant drafts review notes. It does not decide, accuse, freeze accounts, or move money.",
			"A local stub is used when Gemini is not configured. Stub output is labeled live=false.",
			"This is not a regulatory certification or a measured fraud-detection accuracy claim.",
		},
	}
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	p, ok := s.require(w, r, "read")
	if !ok {
		return
	}
	sc, err := s.Store.Begin(r.Context(), p.TenantID)
	if err != nil {
		s.writeDomain(w, r, err)
		return
	}
	defer sc.Rollback()
	counts, err := sc.Counts(r.Context())
	if err != nil {
		s.writeDomain(w, r, err)
		return
	}
	cases, err := sc.ListCases(r.Context(), "", time.Time{}, "", 8)
	if err != nil {
		s.writeDomain(w, r, err)
		return
	}
	depth, err := s.Store.QueueDepth(r.Context(), p.TenantID)
	if err != nil {
		s.writeDomain(w, r, err)
		return
	}
	_ = sc.Commit(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"counts": counts, "cases": cases, "queue_depth": depth, "meta": s.metaBody()})
}

func (s *Server) ingestOne(w http.ResponseWriter, r *http.Request) {
	p, ok := s.require(w, r, "ingest")
	if !ok {
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.fail(w, r, http.StatusRequestEntityTooLarge, "body_too_large", "Request body exceeds the limit")
		return
	}
	var event ingestion.Event
	if err := json.Unmarshal(body, &event); err != nil {
		_ = s.Ingest.QuarantineBytes(r.Context(), p, body, "invalid_json")
		s.Metrics.IngestQuarantine.Add(1)
		s.fail(w, r, http.StatusBadRequest, "invalid_json", "Event JSON could not be parsed and was quarantined")
		return
	}
	res, err := s.Ingest.Ingest(r.Context(), p, event)
	s.countIngest(res)
	if err != nil && res.Status == "" {
		s.writeDomain(w, r, err)
		return
	}
	status := http.StatusOK
	switch res.Status {
	case "quarantined":
		status = http.StatusUnprocessableEntity
	case "conflict":
		status = http.StatusConflict
	}
	writeJSON(w, status, res)
}

func (s *Server) ingestBatch(w http.ResponseWriter, r *http.Request) {
	p, ok := s.require(w, r, "ingest")
	if !ok {
		return
	}
	var body struct {
		Events []ingestion.Event `json:"events"`
	}
	if err := decode(r, &body); err != nil {
		s.fail(w, r, http.StatusBadRequest, "invalid_json", "Batch body must be JSON")
		return
	}
	if len(body.Events) == 0 || len(body.Events) > 100 {
		s.fail(w, r, http.StatusUnprocessableEntity, "invalid", "Batch must contain 1 to 100 events")
		return
	}
	results := make([]ingestion.Result, 0, len(body.Events))
	for _, event := range body.Events {
		res, err := s.Ingest.Ingest(r.Context(), p, event)
		if err != nil && res.Status == "" {
			res = ingestion.Result{IdempotencyKey: event.IdempotencyKey, Status: "error", QuarantineReason: "internal"}
		}
		s.countIngest(res)
		results = append(results, res)
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

func (s *Server) countIngest(res ingestion.Result) {
	switch res.Status {
	case "accepted":
		s.Metrics.IngestAccepted.Add(1)
	case "replayed":
		s.Metrics.IngestReplay.Add(1)
	case "quarantined":
		s.Metrics.IngestQuarantine.Add(1)
	}
}

func (s *Server) transactions(w http.ResponseWriter, r *http.Request) {
	p, ok := s.require(w, r, "read")
	if !ok {
		return
	}
	limit := queryLimit(r, 50)
	sc, err := s.Store.Begin(r.Context(), p.TenantID)
	if err != nil {
		s.writeDomain(w, r, err)
		return
	}
	defer sc.Rollback()
	rows, err := sc.ListTransactions(r.Context(), limit)
	if err != nil {
		s.writeDomain(w, r, err)
		return
	}
	_ = sc.Commit(r.Context())
	if rows == nil {
		rows = []domain.Transaction{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": rows})
}

func (s *Server) listCases(w http.ResponseWriter, r *http.Request) {
	p, ok := s.require(w, r, "read")
	if !ok {
		return
	}
	limit := queryLimit(r, 25)
	cursor, cursorID, err := decodeCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		s.fail(w, r, http.StatusUnprocessableEntity, "invalid", "Cursor is invalid")
		return
	}
	sc, err := s.Store.Begin(r.Context(), p.TenantID)
	if err != nil {
		s.writeDomain(w, r, err)
		return
	}
	defer sc.Rollback()
	rows, err := sc.ListCases(r.Context(), r.URL.Query().Get("status"), cursor, cursorID, limit)
	if err != nil {
		s.writeDomain(w, r, err)
		return
	}
	_ = sc.Commit(r.Context())
	next := ""
	if len(rows) == limit {
		last := rows[len(rows)-1]
		next = encodeCursor(last.OpenedAt, last.ID)
	}
	if rows == nil {
		rows = []domain.Case{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": rows, "page": map[string]any{"next_cursor": next, "limit": limit}})
}

func (s *Server) getCase(w http.ResponseWriter, r *http.Request) {
	p, ok := s.require(w, r, "read")
	if !ok {
		return
	}
	sc, err := s.Store.Begin(r.Context(), p.TenantID)
	if err != nil {
		s.writeDomain(w, r, err)
		return
	}
	defer sc.Rollback()
	id := r.PathValue("id")
	cse, err := sc.GetCase(r.Context(), id)
	if err != nil {
		s.writeDomain(w, r, err)
		return
	}
	signals, err := sc.ListSignals(r.Context(), id)
	if err != nil {
		s.writeDomain(w, r, err)
		return
	}
	txns, err := sc.CaseTransactions(r.Context(), id)
	if err != nil {
		s.writeDomain(w, r, err)
		return
	}
	notes, err := sc.ListNotes(r.Context(), id)
	if err != nil {
		s.writeDomain(w, r, err)
		return
	}
	_ = sc.Commit(r.Context())
	if signals == nil {
		signals = []domain.Signal{}
	}
	if txns == nil {
		txns = []domain.Transaction{}
	}
	if notes == nil {
		notes = []domain.Note{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"case": cse, "signals": signals, "transactions": txns, "notes": notes})
}

func (s *Server) addNote(w http.ResponseWriter, r *http.Request) {
	p, ok := s.require(w, r, "note")
	if !ok {
		return
	}
	var body struct {
		Body string `json:"body"`
	}
	if err := decode(r, &body); err != nil {
		s.fail(w, r, http.StatusBadRequest, "invalid_json", "Request body must be JSON")
		return
	}
	note, err := s.Cases.AddNote(r.Context(), p, r.PathValue("id"), body.Body)
	if err != nil {
		s.writeDomain(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, note)
}

func (s *Server) act(w http.ResponseWriter, r *http.Request) {
	p, ok := s.require(w, r, "read")
	if !ok {
		return
	}
	var body casesvc.Action
	if err := decode(r, &body); err != nil {
		s.fail(w, r, http.StatusBadRequest, "invalid_json", "Request body must be JSON")
		return
	}
	updated, err := s.Cases.Act(r.Context(), p, r.PathValue("id"), body)
	if err != nil {
		s.writeDomain(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) evidence(w http.ResponseWriter, r *http.Request) {
	s.withCaseRead(w, r, func(ctx context.Context, sc *store.Scope, id string) (any, error) {
		return sc.ListEvidence(ctx, id)
	})
}

func (s *Server) listReports(w http.ResponseWriter, r *http.Request) {
	s.withCaseRead(w, r, func(ctx context.Context, sc *store.Scope, id string) (any, error) {
		return sc.ListReports(ctx, id)
	})
}

func (s *Server) audit(w http.ResponseWriter, r *http.Request) {
	s.withCaseRead(w, r, func(ctx context.Context, sc *store.Scope, id string) (any, error) {
		return sc.ListAudit(ctx, "case", id)
	})
}

func (s *Server) listAssistant(w http.ResponseWriter, r *http.Request) {
	s.withCaseRead(w, r, func(ctx context.Context, sc *store.Scope, id string) (any, error) {
		return sc.ListAssistant(ctx, id, time.Now().UTC())
	})
}

func (s *Server) withCaseRead(w http.ResponseWriter, r *http.Request, fn func(context.Context, *store.Scope, string) (any, error)) {
	p, ok := s.require(w, r, "read")
	if !ok {
		return
	}
	sc, err := s.Store.Begin(r.Context(), p.TenantID)
	if err != nil {
		s.writeDomain(w, r, err)
		return
	}
	defer sc.Rollback()
	id := r.PathValue("id")
	if _, err := sc.GetCase(r.Context(), id); err != nil {
		s.writeDomain(w, r, err)
		return
	}
	data, err := fn(r.Context(), sc, id)
	if err != nil {
		s.writeDomain(w, r, err)
		return
	}
	_ = sc.Commit(r.Context())
	if data == nil {
		data = []any{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (s *Server) createReport(w http.ResponseWriter, r *http.Request) {
	p, ok := s.require(w, r, "report")
	if !ok {
		return
	}
	start := time.Now()
	row, err := s.Reports.Generate(r.Context(), p, r.PathValue("id"))
	s.Metrics.ObserveAI(time.Since(start), err == nil && row.Status == "ready")
	if row.ID != "" {
		writeJSON(w, http.StatusOK, row)
		return
	}
	s.writeDomain(w, r, err)
}

func (s *Server) caseGraph(w http.ResponseWriter, r *http.Request) {
	p, ok := s.require(w, r, "read")
	if !ok {
		return
	}
	g, err := s.Graph.ForCase(r.Context(), p.TenantID, r.PathValue("id"))
	if err != nil {
		s.writeDomain(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (s *Server) knockouts(w http.ResponseWriter, r *http.Request) {
	p, ok := s.require(w, r, "read")
	if !ok {
		return
	}
	if s.Knockouts == nil {
		s.fail(w, r, http.StatusInternalServerError, "internal", "Internal error")
		return
	}
	board, err := s.Knockouts.ForCase(r.Context(), p.TenantID, r.PathValue("id"))
	if err != nil {
		s.writeDomain(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, board)
}

func (s *Server) dissent(w http.ResponseWriter, r *http.Request) {
	p, ok := s.require(w, r, "ask")
	if !ok {
		return
	}
	out, err := s.Reports.Dissent(r.Context(), p, r.PathValue("id"))
	if err != nil {
		s.writeDomain(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) ask(w http.ResponseWriter, r *http.Request) {
	p, ok := s.require(w, r, "ask")
	if !ok {
		return
	}
	var body struct {
		Question string `json:"question"`
	}
	if err := decode(r, &body); err != nil {
		s.fail(w, r, http.StatusBadRequest, "invalid_json", "Request body must be JSON")
		return
	}
	msg, err := s.Reports.Ask(r.Context(), p, r.PathValue("id"), body.Question)
	if err != nil {
		s.writeDomain(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, msg)
}

func (s *Server) replayJob(w http.ResponseWriter, r *http.Request) {
	p, ok := s.require(w, r, "replay_job")
	if !ok {
		return
	}
	if err := s.Store.ReplayJob(r.Context(), p.TenantID, r.PathValue("id")); err != nil {
		s.writeDomain(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "pending"})
}

func (s *Server) retention(w http.ResponseWriter, r *http.Request) {
	p, ok := s.require(w, r, "retention")
	if !ok {
		return
	}
	sc, err := s.Store.Begin(r.Context(), p.TenantID)
	if err != nil {
		s.writeDomain(w, r, err)
		return
	}
	defer sc.Rollback()
	n, err := sc.PurgeAssistant(r.Context(), time.Now().UTC())
	if err != nil {
		s.writeDomain(w, r, err)
		return
	}
	if err := sc.InsertAudit(r.Context(), domain.AuditEvent{
		ID: idgen.New("aud_"), ActorID: p.UserID, ActorLabel: p.Email, Action: "assistant_purged",
		EntityType: "tenant", EntityID: p.TenantID, Metadata: map[string]any{"deleted": n},
	}); err != nil {
		s.writeDomain(w, r, err)
		return
	}
	if err := sc.Commit(r.Context()); err != nil {
		s.writeDomain(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": n})
}

func (s *Server) metrics(w http.ResponseWriter, r *http.Request) {
	p, ok := s.require(w, r, "metrics")
	if !ok {
		return
	}
	depth, err := s.Store.QueueDepth(r.Context(), p.TenantID)
	if err != nil {
		s.writeDomain(w, r, err)
		return
	}
	snap := s.Metrics.Snapshot()
	snap["queue_depth"] = int64(depth)
	writeJSON(w, http.StatusOK, map[string]any{
		"metrics": snap,
		"note":    "Counters are for this process only. They are not a capacity or accuracy claim.",
	})
}

func (s *Server) require(w http.ResponseWriter, r *http.Request, action string) (domain.Principal, bool) {
	s.Metrics.Requests.Add(1)
	if r.Method == http.MethodPost && !csrf(r) {
		s.fail(w, r, http.StatusForbidden, "csrf", "Missing X-FinGuard-Request header")
		return domain.Principal{}, false
	}
	p, err := s.Auth.FromToken(r.Context(), tokenFrom(r))
	if err != nil {
		s.writeDomain(w, r, domain.ErrUnauthenticated)
		return domain.Principal{}, false
	}
	if !auth.Can(p.Role, action) {
		s.writeDomain(w, r, domain.ErrForbidden)
		return domain.Principal{}, false
	}
	return p, true
}

func (s *Server) writeDomain(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrUnauthenticated):
		s.fail(w, r, http.StatusUnauthorized, "unauthenticated", "Authentication is required")
	case errors.Is(err, domain.ErrForbidden):
		s.fail(w, r, http.StatusForbidden, "forbidden", "This role cannot perform that action")
	case errors.Is(err, domain.ErrNotFound):
		s.fail(w, r, http.StatusNotFound, "not_found", "Record not found")
	case errors.Is(err, domain.ErrVersion):
		s.fail(w, r, http.StatusConflict, "version_conflict", "The case changed. Refresh and try again")
	case errors.Is(err, domain.ErrConflict):
		s.fail(w, r, http.StatusConflict, "conflict", "Idempotency key was reused with a different payload")
	case errors.Is(err, domain.ErrRationale):
		s.fail(w, r, http.StatusUnprocessableEntity, "invalid", "A rationale of 20 to 2000 characters is required")
	case errors.Is(err, domain.ErrInvalid):
		s.fail(w, r, http.StatusUnprocessableEntity, "invalid", "The request is not valid for the current case state")
	default:
		s.Log.Error("handler error", "request_id", httpx.RequestID(r.Context()), "error", err.Error())
		s.fail(w, r, http.StatusInternalServerError, "internal", "Internal error")
	}
}

func (s *Server) fail(w http.ResponseWriter, r *http.Request, status int, code, msg string) {
	writeJSON(w, status, map[string]any{"error": map[string]any{
		"code": code, "message": msg, "request_id": httpx.RequestID(r.Context()),
	}})
}

func csrf(r *http.Request) bool {
	return r.Header.Get("X-FinGuard-Request") == "1"
}

func tokenFrom(r *http.Request) string {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	c, err := r.Cookie("finguard_session")
	if err != nil {
		return ""
	}
	return c.Value
}

func decode(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func publicUser(p domain.Principal) map[string]any {
	return map[string]any{"id": p.UserID, "email": p.Email, "name": p.Name, "role": p.Role, "tenant_id": p.TenantID}
}

func queryLimit(r *http.Request, def int) int {
	n, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if n <= 0 {
		return def
	}
	if n > 100 {
		return 100
	}
	return n
}

func encodeCursor(t time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(t.UTC().Format(time.RFC3339Nano) + "|" + id))
}

func decodeCursor(raw string) (time.Time, string, error) {
	if raw == "" {
		return time.Time{}, "", nil
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return time.Time{}, "", err
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", errors.New("cursor")
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	return t, parts[1], err
}

func (s *Server) static(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		s.fail(w, r, http.StatusNotFound, "not_found", "Unknown API route")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		s.fail(w, r, http.StatusMethodNotAllowed, "method", "Method not allowed")
		return
	}
	sub, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/")
	if name == "" {
		name = "index.html"
	}
	if _, err := fs.Stat(sub, name); err != nil {
		if strings.HasPrefix(name, "assets/") || strings.HasPrefix(name, "images/") {
			http.NotFound(w, r)
			return
		}
		name = "index.html"
	}
	if name == "index.html" {
		content, err := fs.ReadFile(sub, name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(content))
		return
	}
	http.FileServer(http.FS(sub)).ServeHTTP(w, r)
}
