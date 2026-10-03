// Package aigateway calls a model provider with no tools and validates the
// untrusted text it returns. The stub provider is a local fallback, not a
// claim that Gemini ran.
package aigateway

import "time"

const (
	ReportPromptVersion    = "finguard-report-v1"
	AssistantPromptVersion = "finguard-assistant-v1"
	DissentPromptVersion   = "finguard-dissent-v1"
)

type EvidenceItem struct {
	ID         string    `json:"id"`
	Kind       string    `json:"kind"`
	Summary    string    `json:"summary"`
	OccurredAt time.Time `json:"occurred_at,omitempty"`
}

type SignalView struct {
	RuleCode      string   `json:"rule_code"`
	RuleVersion   string   `json:"rule_version"`
	Explanation   string   `json:"explanation"`
	Threshold     string   `json:"threshold"`
	ObservedValue string   `json:"observed_value"`
	EvidenceIDs   []string `json:"evidence_ids"`
}

// Package is the only case context a provider may see.
type Package struct {
	CaseID      string         `json:"case_id"`
	Items       []EvidenceItem `json:"items"`
	Signals     []SignalView   `json:"signals"`
	Devices     []string       `json:"devices"`
	Truncated   bool           `json:"truncated"`
	Limitations []string       `json:"package_limitations,omitempty"`
}

type Request struct {
	PromptVersion   string
	System          string
	User            string
	MaxOutputTokens int
	JSONMode        bool
	EvidenceIDs     []string
}

type Response struct {
	Text         string
	Model        string
	Provider     string
	Live         bool
	InputTokens  int
	OutputTokens int
	Latency      time.Duration
}

type Fact struct {
	Claim       string   `json:"claim"`
	EvidenceIDs []string `json:"evidence_ids"`
}

type Hypothesis struct {
	Claim       string   `json:"claim"`
	EvidenceIDs []string `json:"evidence_ids"`
	Uncertainty string   `json:"uncertainty"`
}

type Unverified struct {
	Claim  string `json:"claim"`
	Reason string `json:"reason"`
}

// Report is the validated investigation draft. Model-supplied provider
// metadata is discarded and replaced by the server.
type Report struct {
	Summary                 string         `json:"summary"`
	ObservedFacts           []Fact         `json:"observed_facts"`
	Hypotheses              []Hypothesis   `json:"hypotheses"`
	AlternativeExplanations []Hypothesis   `json:"alternative_explanations"`
	MissingEvidence         []string       `json:"missing_evidence"`
	SuggestedReviewSteps    []string       `json:"suggested_review_steps"`
	Limitations             []string       `json:"limitations"`
	UnverifiedClaims        []Unverified   `json:"unverified_claims"`
	ValidationStatus        string         `json:"validation_status"`
	ModelMetadata           map[string]any `json:"model_metadata"`
}

// Dissent is a model reading of computed knockouts. It may not add facts.
type Dissent struct {
	Concern          Fact         `json:"concern"`
	Benign           Fact         `json:"benign"`
	WhatWouldChange  []Fact       `json:"what_would_change"`
	Limitations      []string     `json:"limitations"`
	UnverifiedClaims []Unverified `json:"unverified_claims"`
	ValidationStatus string       `json:"validation_status"`
}

type AssistantAnswer struct {
	Answer        string   `json:"answer"`
	Citations     []string `json:"citations"`
	Uncertainty   string   `json:"uncertainty"`
	Refused       bool     `json:"refused"`
	RefusalReason string   `json:"refusal_reason"`
	Limitations   []string `json:"limitations"`
}
