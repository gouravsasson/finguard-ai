package report

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"finguard.ai/internal/aigateway"
	"finguard.ai/internal/domain"
	"finguard.ai/internal/evidence"
	"finguard.ai/internal/idgen"
	"finguard.ai/internal/knockout"
	"finguard.ai/internal/store"
)

type Service struct {
	Store        *store.Store
	Provider     aigateway.Generator
	ProviderName string
	Timeout      time.Duration
	MaxChars     int
	MaxTokens    int
	Retention    time.Duration
	budget       budget
}

func (s *Service) ConfigureBudget(max int) {
	s.budget.max = max
}

type budget struct {
	mu  sync.Mutex
	day string
	n   int
	max int
}

func (b *budget) allow() bool {
	if b.max <= 0 {
		return true
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	day := time.Now().UTC().Format("2006-01-02")
	if b.day != day {
		b.day = day
		b.n = 0
	}
	if b.n >= b.max {
		return false
	}
	b.n++
	return true
}

func (s *Service) Generate(ctx context.Context, p domain.Principal, caseID string) (domain.AIReport, error) {
	sc, err := s.Store.Begin(ctx, p.TenantID)
	if err != nil {
		return domain.AIReport{}, err
	}
	pkg, err := evidence.Build(ctx, sc, caseID)
	if err != nil {
		sc.Rollback()
		return domain.AIReport{}, err
	}
	if _, err := sc.GetCase(ctx, caseID); err != nil {
		sc.Rollback()
		return domain.AIReport{}, err
	}
	sc.Rollback()
	pkg = s.bound(pkg)
	return s.complete(ctx, p, caseID, pkg)
}

func (s *Service) complete(ctx context.Context, p domain.Principal, caseID string, pkg aigateway.Package) (domain.AIReport, error) {
	system, user, err := aigateway.BuildReportPrompt(pkg)
	if err != nil {
		return domain.AIReport{}, err
	}
	if !s.budget.allow() {
		return s.fail(ctx, p, caseID, "daily AI call cap reached", false)
	}
	timeout := s.Timeout
	if timeout == 0 {
		timeout = 20 * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	resp, err := s.Provider.Generate(callCtx, aigateway.Request{
		PromptVersion: aigateway.ReportPromptVersion, System: system, User: user,
		MaxOutputTokens: s.MaxTokens, JSONMode: true, EvidenceIDs: aigateway.EvidenceIDs(pkg),
	})
	if err != nil {
		poison := errors.Is(err, aigateway.ErrPoison)
		return s.fail(ctx, p, caseID, err.Error(), poison)
	}
	meta := map[string]any{
		"provider": resp.Provider, "model": resp.Model, "prompt_version": aigateway.ReportPromptVersion,
		"live": resp.Live, "generated_at": time.Now().UTC().Format(time.RFC3339),
	}
	validated, err := aigateway.ValidateReport(resp.Text, aigateway.EvidenceIDs(pkg), meta)
	if err != nil {
		poison := errors.Is(err, aigateway.ErrPoison)
		return s.fail(ctx, p, caseID, err.Error(), poison)
	}
	if pkg.Truncated {
		validated.Limitations = append(validated.Limitations, "The evidence package was truncated before the model call.")
	}
	body, err := decodeReport(validated)
	if err != nil {
		return domain.AIReport{}, err
	}
	row := domain.AIReport{
		ID: idgen.New("rep_"), CaseID: caseID, Status: "ready", ValidationStatus: validated.ValidationStatus,
		Provider: resp.Provider, Model: resp.Model, PromptVersion: aigateway.ReportPromptVersion, Live: resp.Live,
		LatencyMS: int(resp.Latency.Milliseconds()), InputTokens: resp.InputTokens, OutputTokens: resp.OutputTokens,
		Report: body,
	}
	if err := s.persist(ctx, p, row, "ai_report_ready"); err != nil {
		return domain.AIReport{}, err
	}
	return row, nil
}

func (s *Service) fail(ctx context.Context, p domain.Principal, caseID, message string, poison bool) (domain.AIReport, error) {
	status := "failed"
	validation := "unavailable"
	action := "ai_report_failed"
	ret := errors.New(message)
	if poison {
		status = "rejected"
		validation = "rejected"
		action = "ai_report_rejected"
		ret = aigateway.ErrPoison
		message = "model output was rejected because it requested a tool or control action"
	}
	row := domain.AIReport{
		ID: idgen.New("rep_"), CaseID: caseID, Status: status, ValidationStatus: validation,
		Provider: s.name(), Model: "", PromptVersion: aigateway.ReportPromptVersion, Live: false,
		Report: map[string]any{
			"summary":        "A report could not be produced. The case remains open for human review.",
			"limitations":    []string{message},
			"observed_facts": []any{},
			"model_metadata": map[string]any{"live": false, "prompt_version": aigateway.ReportPromptVersion},
		},
	}
	if err := s.persist(ctx, p, row, action); err != nil {
		return domain.AIReport{}, err
	}
	return row, ret
}

func (s *Service) persist(ctx context.Context, p domain.Principal, row domain.AIReport, action string) error {
	sc, err := s.Store.Begin(ctx, p.TenantID)
	if err != nil {
		return err
	}
	defer sc.Rollback()
	if err := sc.InsertReport(ctx, row); err != nil {
		return err
	}
	if err := sc.InsertAudit(ctx, domain.AuditEvent{
		ID: idgen.New("aud_"), ActorID: p.UserID, ActorLabel: actorLabel(p), Action: action,
		EntityType: "case", EntityID: row.CaseID,
		Metadata: map[string]any{"provider": row.Provider, "live": row.Live, "validation_status": row.ValidationStatus, "status": row.Status},
	}); err != nil {
		return err
	}
	return sc.Commit(ctx)
}

func (s *Service) Ask(ctx context.Context, p domain.Principal, caseID, question string) (domain.AssistantMessage, error) {
	question = strings.TrimSpace(question)
	if len(question) > 1000 {
		return domain.AssistantMessage{}, domain.ErrInvalid
	}
	sc, err := s.Store.Begin(ctx, p.TenantID)
	if err != nil {
		return domain.AssistantMessage{}, err
	}
	if _, err := sc.GetCase(ctx, caseID); err != nil {
		sc.Rollback()
		return domain.AssistantMessage{}, err
	}
	pkg, err := evidence.Build(ctx, sc, caseID)
	if err != nil {
		sc.Rollback()
		return domain.AssistantMessage{}, err
	}
	history, err := sc.ListAssistant(ctx, caseID, time.Now().UTC())
	sc.Rollback()
	if err != nil {
		return domain.AssistantMessage{}, err
	}
	retain := time.Now().UTC().Add(s.retention())
	userMsg := domain.AssistantMessage{
		ID: idgen.New("msg_"), CaseID: caseID, AuthorID: p.UserID, Role: "investigator",
		Content: question, Citations: []string{}, RetainUntil: retain,
	}
	if refuse, reason := aigateway.RefuseQuestion(question, caseID); refuse {
		answer := domain.AssistantMessage{
			ID: idgen.New("msg_"), CaseID: caseID, Role: "assistant", Content: reason,
			Citations: []string{}, Refused: true, RetainUntil: retain,
		}
		if err := s.savePair(ctx, p, userMsg, answer); err != nil {
			return domain.AssistantMessage{}, err
		}
		return answer, nil
	}
	var lines []string
	if len(history) > 6 {
		history = history[len(history)-6:]
	}
	for _, m := range history {
		lines = append(lines, m.Role+": "+clip(m.Content, 240))
	}
	pkg = s.bound(pkg)
	system, user, err := aigateway.BuildAssistantPrompt(pkg, question, lines)
	if err != nil {
		return domain.AssistantMessage{}, err
	}
	if !s.budget.allow() {
		answer := domain.AssistantMessage{
			ID: idgen.New("msg_"), CaseID: caseID, Role: "assistant", Refused: true, RetainUntil: retain,
			Content: "The assistant is not available because the daily call cap was reached. The case evidence is still available.",
		}
		if err := s.savePair(ctx, p, userMsg, answer); err != nil {
			return domain.AssistantMessage{}, err
		}
		return answer, nil
	}
	timeout := s.Timeout
	if timeout == 0 {
		timeout = 20 * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	resp, err := s.Provider.Generate(callCtx, aigateway.Request{
		PromptVersion: aigateway.AssistantPromptVersion, System: system, User: user,
		MaxOutputTokens: s.MaxTokens, JSONMode: true, EvidenceIDs: aigateway.EvidenceIDs(pkg),
	})
	answer := domain.AssistantMessage{ID: idgen.New("msg_"), CaseID: caseID, Role: "assistant", RetainUntil: retain}
	if err != nil {
		answer.Refused = true
		answer.Content = "The assistant could not answer because the model call failed. Case evidence is unchanged."
	} else {
		parsed, vErr := aigateway.ValidateAnswer(resp.Text, aigateway.EvidenceIDs(pkg))
		if vErr != nil {
			answer.Refused = true
			answer.Content = "The draft answer was rejected because it was not valid for this case."
		} else {
			answer.Content = parsed.Answer
			answer.Citations = parsed.Citations
			answer.Refused = parsed.Refused
		}
	}
	if err := s.savePair(ctx, p, userMsg, answer); err != nil {
		return domain.AssistantMessage{}, err
	}
	return answer, nil
}

func (s *Service) savePair(ctx context.Context, p domain.Principal, user, answer domain.AssistantMessage) error {
	sc, err := s.Store.Begin(ctx, p.TenantID)
	if err != nil {
		return err
	}
	defer sc.Rollback()
	if _, err := sc.GetCase(ctx, user.CaseID); err != nil {
		return err
	}
	if err := sc.InsertAssistant(ctx, user); err != nil {
		return err
	}
	if err := sc.InsertAssistant(ctx, answer); err != nil {
		return err
	}
	return sc.Commit(ctx)
}

func (s *Service) bound(pkg aigateway.Package) aigateway.Package {
	max := s.MaxChars
	if max <= 0 {
		max = 24000
	}
	raw, err := json.Marshal(pkg)
	if err != nil || len(raw) <= max {
		return pkg
	}
	for len(pkg.Items) > 1 {
		pkg.Items = pkg.Items[:len(pkg.Items)-1]
		pkg.Truncated = true
		raw, _ = json.Marshal(pkg)
		if len(raw) <= max {
			break
		}
	}
	pkg.Truncated = true
	return pkg
}

// Dissent is the model reading of a knockout board. A failed or rejected model
// call returns the deterministic reading instead of an empty desk.
type Dissent struct {
	Provider         string                 `json:"provider"`
	Model            string                 `json:"model"`
	Live             bool                   `json:"live"`
	PromptVersion    string                 `json:"prompt_version"`
	ValidationStatus string                 `json:"validation_status"`
	Concern          aigateway.Fact         `json:"concern"`
	Benign           aigateway.Fact         `json:"benign"`
	WhatWouldChange  []aigateway.Fact       `json:"what_would_change"`
	Limitations      []string               `json:"limitations"`
	Unverified       []aigateway.Unverified `json:"unverified_claims"`
}

func (s *Service) Dissent(ctx context.Context, p domain.Principal, caseID string) (Dissent, error) {
	board, err := (&knockout.Service{Store: s.Store}).ForCase(ctx, p.TenantID, caseID)
	if err != nil {
		return Dissent{}, err
	}
	fallback := dissentFromReading(board)
	allowed := boardEvidence(board)
	if len(allowed) == 0 || !s.budget.allow() {
		fallback.Limitations = append(fallback.Limitations, "The model was not called. This text is the deterministic replay reading.")
		return fallback, nil
	}
	payload, err := json.Marshal(board)
	if err != nil {
		return fallback, nil
	}
	system, user, err := aigateway.BuildDissentPrompt(payload)
	if err != nil {
		return fallback, nil
	}
	timeout := s.Timeout
	if timeout == 0 {
		timeout = 20 * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	resp, err := s.Provider.Generate(callCtx, aigateway.Request{
		PromptVersion: aigateway.DissentPromptVersion, System: system, User: user,
		MaxOutputTokens: s.MaxTokens, JSONMode: true, EvidenceIDs: allowed,
	})
	if err != nil {
		fallback.Limitations = append(fallback.Limitations, "The model call failed. The deterministic replay reading is shown instead.")
		return fallback, nil
	}
	parsed, err := aigateway.ValidateDissent(resp.Text, allowed)
	if err != nil {
		fallback.Limitations = append(fallback.Limitations, "The model dissent was rejected. The deterministic replay reading is shown instead.")
		return fallback, nil
	}
	return Dissent{
		Provider: resp.Provider, Model: resp.Model, Live: resp.Live, PromptVersion: aigateway.DissentPromptVersion,
		ValidationStatus: parsed.ValidationStatus, Concern: parsed.Concern, Benign: parsed.Benign,
		WhatWouldChange: parsed.WhatWouldChange, Limitations: parsed.Limitations, Unverified: parsed.UnverifiedClaims,
	}, nil
}

func dissentFromReading(board knockout.Board) Dissent {
	concernIDs := board.Reading.ConcernEvidence
	benignIDs := board.Reading.BenignEvidence
	var changes []aigateway.Fact
	for _, line := range board.Reading.WouldChange {
		changes = append(changes, aigateway.Fact{Claim: line, EvidenceIDs: board.Reading.ChangeEvidence})
	}
	if changes == nil {
		changes = []aigateway.Fact{}
	}
	return Dissent{
		Provider: "deterministic-replay", Model: "", Live: false, PromptVersion: aigateway.DissentPromptVersion,
		ValidationStatus: "deterministic",
		Concern:          aigateway.Fact{Claim: board.Reading.Concern, EvidenceIDs: concernIDs},
		Benign:           aigateway.Fact{Claim: board.Reading.Benign, EvidenceIDs: benignIDs},
		WhatWouldChange:  changes,
		Limitations: []string{
			"Hypothetical replays do not change stored transactions.",
			"A cleared replay is not proof that the activity was legitimate.",
		},
		Unverified: []aigateway.Unverified{},
	}
}

func boardEvidence(board knockout.Board) []string {
	var ids []string
	ids = append(ids, board.Reading.ConcernEvidence...)
	ids = append(ids, board.Reading.BenignEvidence...)
	ids = append(ids, board.Reading.ChangeEvidence...)
	for _, sig := range board.Signals {
		for _, attempt := range sig.Attempts {
			ids = append(ids, attempt.EvidenceIDs...)
		}
	}
	return knockoutUnique(ids)
}

func knockoutUnique(ids []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func (s *Service) retention() time.Duration {
	if s.Retention == 0 {
		return 30 * 24 * time.Hour
	}
	return s.Retention
}

func decodeReport(r aigateway.Report) (map[string]any, error) {
	b, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	err = json.Unmarshal(b, &m)
	return m, err
}

func actorLabel(p domain.Principal) string {
	if p.Email != "" {
		return p.Email
	}
	return "system"
}

func (s *Service) name() string {
	if s.ProviderName != "" {
		return s.ProviderName
	}
	return "local-stub"
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
