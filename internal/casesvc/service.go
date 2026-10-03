package casesvc

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"finguard.ai/internal/auth"
	"finguard.ai/internal/domain"
	"finguard.ai/internal/idgen"
	"finguard.ai/internal/store"
)

type Service struct {
	Store *store.Store
}

type Action struct {
	Action    string `json:"action"`
	Rationale string `json:"rationale"`
	Version   int    `json:"version"`
}

var transitions = map[string]map[string]string{
	domain.StatusOpen:      {"start_review": domain.StatusInReview, "escalate": domain.StatusEscalated},
	domain.StatusInReview:  {"escalate": domain.StatusEscalated, "resolve": domain.StatusResolved},
	domain.StatusEscalated: {"resolve": domain.StatusResolved},
	domain.StatusResolved:  {"close": domain.StatusClosed},
}

func (s *Service) Act(ctx context.Context, p domain.Principal, caseID string, in Action) (domain.Case, error) {
	perm := map[string]string{
		"assign": "review", "start_review": "review", "escalate": "escalate",
		"resolve": "resolve", "close": "close",
	}[in.Action]
	if perm == "" || !auth.Can(p.Role, perm) {
		return domain.Case{}, domain.ErrForbidden
	}
	rationale := strings.TrimSpace(in.Rationale)
	if in.Action == "escalate" || in.Action == "resolve" || in.Action == "close" {
		if utf8.RuneCountInString(rationale) < 20 || utf8.RuneCountInString(rationale) > 2000 {
			return domain.Case{}, domain.ErrRationale
		}
	}
	sc, err := s.Store.Begin(ctx, p.TenantID)
	if err != nil {
		return domain.Case{}, err
	}
	defer sc.Rollback()
	current, err := sc.GetCase(ctx, caseID)
	if err != nil {
		return domain.Case{}, err
	}
	if current.Version != in.Version {
		return domain.Case{}, domain.ErrVersion
	}
	next := current
	switch in.Action {
	case "assign":
		next.AssigneeID = p.UserID
		if current.Status == domain.StatusOpen {
			next.Status = domain.StatusInReview
		}
	default:
		dest, ok := transitions[current.Status][in.Action]
		if !ok {
			return domain.Case{}, domain.ErrInvalid
		}
		next.Status = dest
		if in.Action == "start_review" && next.AssigneeID == "" {
			next.AssigneeID = p.UserID
		}
	}
	if next.Status == domain.StatusResolved || next.Status == domain.StatusClosed {
		now := time.Now().UTC()
		next.ResolvedAt = &now
		if next.Resolution == "" {
			next.Resolution = "no_adverse_action"
		}
		next.Rationale = rationale
	} else if rationale != "" {
		next.Rationale = rationale
	}
	if err := sc.UpdateCase(ctx, next, current.Version); err != nil {
		return domain.Case{}, err
	}
	if err := sc.InsertDecision(ctx, idgen.New("dec_"), caseID, p.UserID, in.Action, current.Status, next.Status, rationale); err != nil {
		return domain.Case{}, err
	}
	if err := sc.InsertAudit(ctx, domain.AuditEvent{
		ID: idgen.New("aud_"), ActorID: p.UserID, ActorLabel: p.Email, Action: "case_" + in.Action,
		EntityType: "case", EntityID: caseID, Rationale: rationale,
		Metadata: map[string]any{"from": current.Status, "to": next.Status, "resolution": next.Resolution},
	}); err != nil {
		return domain.Case{}, err
	}
	updated, err := sc.GetCase(ctx, caseID)
	if err != nil {
		return domain.Case{}, err
	}
	if err := sc.Commit(ctx); err != nil {
		return domain.Case{}, err
	}
	return updated, nil
}

func (s *Service) AddNote(ctx context.Context, p domain.Principal, caseID, body string) (domain.Note, error) {
	if !auth.Can(p.Role, "note") {
		return domain.Note{}, domain.ErrForbidden
	}
	body = strings.TrimSpace(body)
	if body == "" || utf8.RuneCountInString(body) > 4000 {
		return domain.Note{}, domain.ErrInvalid
	}
	sc, err := s.Store.Begin(ctx, p.TenantID)
	if err != nil {
		return domain.Note{}, err
	}
	defer sc.Rollback()
	if _, err := sc.GetCase(ctx, caseID); err != nil {
		return domain.Note{}, err
	}
	note := domain.Note{ID: idgen.New("note_"), CaseID: caseID, AuthorID: p.UserID, Body: body}
	if err := sc.InsertNote(ctx, note); err != nil {
		return domain.Note{}, err
	}
	if err := sc.InsertAudit(ctx, domain.AuditEvent{
		ID: idgen.New("aud_"), ActorID: p.UserID, ActorLabel: p.Email, Action: "note_added",
		EntityType: "case", EntityID: caseID,
	}); err != nil {
		return domain.Note{}, err
	}
	if err := sc.Commit(ctx); err != nil {
		return domain.Note{}, err
	}
	note.CreatedAt = time.Now().UTC()
	return note, nil
}
