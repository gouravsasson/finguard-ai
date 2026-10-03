// Package domain holds shared types for the FinGuard investigation model.
// AI output is untrusted data. Case transitions are human-authorized.
package domain

import (
	"errors"
	"time"
)

const (
	RoleViewer       = "viewer"
	RoleInvestigator = "investigator"
	RoleSenior       = "senior_investigator"
	RoleAdmin        = "admin"

	StatusOpen      = "open"
	StatusInReview  = "in_review"
	StatusEscalated = "escalated"
	StatusResolved  = "resolved"
	StatusClosed    = "closed"

	TriageLow    = "low"
	TriageMedium = "medium"
	TriageHigh   = "high"

	TxnApproved = "approved"
	TxnDeclined = "declined"
	TxnPending  = "pending"

	RawAccepted    = "accepted"
	RawQuarantined = "quarantined"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrForbidden       = errors.New("forbidden")
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrInvalid         = errors.New("invalid")
	ErrRationale       = errors.New("rationale length")
	ErrConflict        = errors.New("conflict")
	ErrVersion         = errors.New("version conflict")
	ErrPoison          = errors.New("poison message")
)

// Principal is the authenticated investigator. TenantID always comes from the
// session, never from a client-supplied header.
type Principal struct {
	UserID   string
	TenantID string
	Email    string
	Name     string
	Role     string
}

// TriageBand is a queue hint derived from signal volume. It is not a fraud score.
func TriageBand(signalCount int) string {
	switch {
	case signalCount >= 3:
		return TriageHigh
	case signalCount >= 1:
		return TriageMedium
	default:
		return TriageLow
	}
}

type Account struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	ExternalID   string    `json:"external_id"`
	DisplayLabel string    `json:"display_label"`
	Currency     string    `json:"currency"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

type Transaction struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	RawEventID      string    `json:"raw_event_id"`
	SourceSystem    string    `json:"source_system"`
	SourceID        string    `json:"source_id"`
	AccountID       string    `json:"account_id"`
	AccountExternal string    `json:"account_external_id,omitempty"`
	MerchantID      string    `json:"merchant_id,omitempty"`
	MerchantName    string    `json:"merchant_name,omitempty"`
	DeviceID        string    `json:"device_id,omitempty"`
	DeviceExternal  string    `json:"device_external_id,omitempty"`
	LocationID      string    `json:"location_id,omitempty"`
	LocationCountry string    `json:"location_country,omitempty"`
	LocationLabel   string    `json:"location_label,omitempty"`
	AmountMinor     int64     `json:"amount_minor"`
	Currency        string    `json:"currency"`
	Status          string    `json:"status"`
	OccurredAt      time.Time `json:"occurred_at"`
	Memo            string    `json:"memo,omitempty"`
	SchemaVersion   string    `json:"schema_version"`
	CreatedAt       time.Time `json:"created_at"`
}

type Signal struct {
	ID            string         `json:"id"`
	TenantID      string         `json:"tenant_id"`
	CaseID        string         `json:"case_id"`
	TransactionID string         `json:"transaction_id"`
	RuleCode      string         `json:"rule_code"`
	RuleVersion   string         `json:"rule_version"`
	Explanation   string         `json:"explanation"`
	Threshold     string         `json:"threshold"`
	ObservedValue string         `json:"observed_value"`
	Inputs        map[string]any `json:"inputs"`
	EvidenceIDs   []string       `json:"evidence_ids"`
	CreatedAt     time.Time      `json:"created_at"`
}

type Case struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	AccountID   string     `json:"account_id"`
	AccountExt  string     `json:"account_external_id,omitempty"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	Title       string     `json:"title"`
	AssigneeID  string     `json:"assignee_id,omitempty"`
	Version     int        `json:"version"`
	SignalCount int        `json:"signal_count"`
	OpenedAt    time.Time  `json:"opened_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	Resolution  string     `json:"resolution,omitempty"`
	Rationale   string     `json:"rationale,omitempty"`
}

type Evidence struct {
	ID             string         `json:"id"`
	TenantID       string         `json:"tenant_id"`
	CaseID         string         `json:"case_id"`
	StableKey      string         `json:"stable_key"`
	Version        int            `json:"version"`
	Kind           string         `json:"kind"`
	Source         string         `json:"source"`
	Classification string         `json:"classification"`
	ContentHash    string         `json:"content_hash"`
	Payload        map[string]any `json:"payload"`
	Provenance     map[string]any `json:"provenance"`
	CreatedAt      time.Time      `json:"created_at"`
	RetentionUntil time.Time      `json:"retention_until"`
}

type AIReport struct {
	ID               string         `json:"id"`
	TenantID         string         `json:"tenant_id"`
	CaseID           string         `json:"case_id"`
	Status           string         `json:"status"`
	ValidationStatus string         `json:"validation_status"`
	Provider         string         `json:"provider"`
	Model            string         `json:"model"`
	PromptVersion    string         `json:"prompt_version"`
	Live             bool           `json:"live"`
	LatencyMS        int            `json:"latency_ms"`
	InputTokens      int            `json:"input_tokens"`
	OutputTokens     int            `json:"output_tokens"`
	Report           map[string]any `json:"report"`
	CreatedAt        time.Time      `json:"created_at"`
}

type Note struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	CaseID    string    `json:"case_id"`
	AuthorID  string    `json:"author_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type AuditEvent struct {
	ID         string         `json:"id"`
	TenantID   string         `json:"tenant_id"`
	ActorID    string         `json:"actor_id,omitempty"`
	ActorLabel string         `json:"actor_label"`
	Action     string         `json:"action"`
	EntityType string         `json:"entity_type"`
	EntityID   string         `json:"entity_id"`
	Rationale  string         `json:"rationale,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

type Job struct {
	ID             string         `json:"id"`
	TenantID       string         `json:"tenant_id"`
	Type           string         `json:"job_type"`
	IdempotencyKey string         `json:"idempotency_key"`
	Payload        map[string]any `json:"payload"`
	Status         string         `json:"status"`
	Attempts       int            `json:"attempts"`
	MaxAttempts    int            `json:"max_attempts"`
	NextAttemptAt  time.Time      `json:"next_attempt_at"`
	LastError      string         `json:"last_error,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type AssistantMessage struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	CaseID      string    `json:"case_id"`
	AuthorID    string    `json:"author_id,omitempty"`
	Role        string    `json:"role"`
	Content     string    `json:"content"`
	Citations   []string  `json:"citations"`
	Refused     bool      `json:"refused"`
	CreatedAt   time.Time `json:"created_at"`
	RetainUntil time.Time `json:"retain_until"`
}

type GraphNode struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Label      string         `json:"label"`
	Kind       string         `json:"kind"`
	Properties map[string]any `json:"properties,omitempty"`
}

type GraphEdge struct {
	ID          string   `json:"id"`
	From        string   `json:"from"`
	To          string   `json:"to"`
	Type        string   `json:"type"`
	Kind        string   `json:"kind"`
	Provenance  string   `json:"provenance"`
	EvidenceIDs []string `json:"evidence_ids"`
}

type Graph struct {
	Nodes       []GraphNode `json:"nodes"`
	Edges       []GraphEdge `json:"edges"`
	Limitations []string    `json:"limitations"`
}
