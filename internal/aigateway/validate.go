package aigateway

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
)

var (
	accusation = regexp.MustCompile(`(?i)\b(criminal|guilty|fraudster|thief)\b`)
	sideEffect = regexp.MustCompile(`(?i)(\bfreeze\b.{0,40}\b(account|card)\b|\bblock\b.{0,40}\b(account|card)\b|\btransfer\b.{0,30}\b(money|funds)\b|\baccuse\b|\bmark (them|him|her|the customer) as (guilty|criminal)\b)`)
)

// ValidateReport parses model text and keeps only claims that cite evidence
// in allowed. Structural failure returns ErrPoison so the caller does not
// treat the text as findings. Unsupported claims are moved, not invented.
func ValidateReport(raw string, allowed []string, meta map[string]any) (Report, error) {
	if hasToolRequest(raw) {
		return Report{}, ErrPoison
	}
	var parsed Report
	if err := json.Unmarshal([]byte(extractJSON(raw)), &parsed); err != nil {
		return Report{}, err
	}
	if strings.TrimSpace(parsed.Summary) == "" {
		return Report{}, errors.New("summary required")
	}
	allow := setOf(allowed)
	var facts []Fact
	var unverified []Unverified
	for _, fact := range parsed.ObservedFacts {
		claim := clip(fact.Claim, 500)
		if claim == "" {
			continue
		}
		ids := filterIDs(fact.EvidenceIDs, allow)
		if accusation.MatchString(claim) || len(ids) == 0 {
			reason := "missing evidence citation"
			if accusation.MatchString(claim) {
				reason = "accusation language is not an observed fact"
			}
			unverified = append(unverified, Unverified{Claim: claim, Reason: reason})
			continue
		}
		facts = append(facts, Fact{Claim: claim, EvidenceIDs: ids})
	}
	hyp, u1 := cleanHypotheses(parsed.Hypotheses, allow)
	alt, u2 := cleanHypotheses(parsed.AlternativeExplanations, allow)
	unverified = append(unverified, u1...)
	unverified = append(unverified, u2...)
	status := "grounded"
	if len(unverified) > 0 || len(facts) == 0 {
		status = "partial"
	}
	if len(facts) == 0 && len(parsed.ObservedFacts) > 0 && len(unverified) > 0 {
		status = "partial"
	}
	limitations := clipList(parsed.Limitations, 12, 300)
	if len(facts) == 0 {
		limitations = append(limitations, "No observed fact remained after evidence-id checks.")
	}
	report := Report{
		Summary:                 clip(parsed.Summary, 2000),
		ObservedFacts:           capFacts(facts, 20),
		Hypotheses:              capHyp(hyp, 12),
		AlternativeExplanations: capHyp(alt, 12),
		MissingEvidence:         clipList(parsed.MissingEvidence, 12, 300),
		SuggestedReviewSteps:    clipList(parsed.SuggestedReviewSteps, 12, 300),
		Limitations:             clipList(limitations, 16, 300),
		UnverifiedClaims:        capUnverified(unverified, 20),
		ValidationStatus:        status,
		ModelMetadata:           meta,
	}
	if report.UnverifiedClaims == nil {
		report.UnverifiedClaims = []Unverified{}
	}
	return report, nil
}

// ValidateDissent keeps only knockout claims that cite allowed evidence.
func ValidateDissent(raw string, allowed []string) (Dissent, error) {
	if hasToolRequest(raw) {
		return Dissent{}, ErrPoison
	}
	var parsed Dissent
	if err := json.Unmarshal([]byte(extractJSON(raw)), &parsed); err != nil {
		return Dissent{}, err
	}
	allow := setOf(allowed)
	var unverified []Unverified
	concern, u1 := oneFact(parsed.Concern, allow)
	benign, u2 := oneFact(parsed.Benign, allow)
	unverified = append(unverified, u1...)
	unverified = append(unverified, u2...)
	var changes []Fact
	for _, fact := range parsed.WhatWouldChange {
		kept, dropped := oneFact(fact, allow)
		unverified = append(unverified, dropped...)
		if kept.Claim != "" {
			changes = append(changes, kept)
		}
	}
	if concern.Claim == "" || benign.Claim == "" {
		return Dissent{}, errors.New("dissent missing a cited claim")
	}
	status := "grounded"
	if len(unverified) > 0 {
		status = "partial"
	}
	if sideEffect.MatchString(concern.Claim) || sideEffect.MatchString(benign.Claim) {
		return Dissent{}, ErrPoison
	}
	return Dissent{
		Concern:          concern,
		Benign:           benign,
		WhatWouldChange:  capFacts(changes, 6),
		Limitations:      clipList(parsed.Limitations, 8, 300),
		UnverifiedClaims: capUnverified(unverified, 12),
		ValidationStatus: status,
	}, nil
}

func oneFact(fact Fact, allow map[string]struct{}) (Fact, []Unverified) {
	claim := clip(fact.Claim, 500)
	if claim == "" {
		return Fact{}, nil
	}
	ids := filterIDs(fact.EvidenceIDs, allow)
	if accusation.MatchString(claim) || len(ids) == 0 {
		reason := "missing evidence citation"
		if accusation.MatchString(claim) {
			reason = "accusation language is not an observed fact"
		}
		return Fact{}, []Unverified{{Claim: claim, Reason: reason}}
	}
	return Fact{Claim: claim, EvidenceIDs: ids}, nil
}

// ValidateAnswer checks assistant citations and strips unknown evidence ids.
func ValidateAnswer(raw string, allowed []string) (AssistantAnswer, error) {
	if hasToolRequest(raw) {
		return AssistantAnswer{}, ErrPoison
	}
	var parsed AssistantAnswer
	if err := json.Unmarshal([]byte(extractJSON(raw)), &parsed); err != nil {
		return AssistantAnswer{}, err
	}
	allow := setOf(allowed)
	var cites []string
	for _, id := range parsed.Citations {
		if _, ok := allow[id]; ok {
			cites = append(cites, id)
		}
	}
	if cites == nil {
		cites = []string{}
	}
	answer := clip(parsed.Answer, 2000)
	limitations := clipList(parsed.Limitations, 8, 300)
	if len(parsed.Citations) > 0 && len(cites) == 0 {
		limitations = append(limitations, "The draft cited evidence ids that are not in this case. Those citations were removed.")
	}
	if strings.TrimSpace(answer) == "" {
		return AssistantAnswer{}, errors.New("answer required")
	}
	uncertainty := parsed.Uncertainty
	switch uncertainty {
	case "low", "medium", "high":
	default:
		uncertainty = "high"
	}
	return AssistantAnswer{
		Answer:        answer,
		Citations:     cites,
		Uncertainty:   uncertainty,
		Refused:       parsed.Refused,
		RefusalReason: clip(parsed.RefusalReason, 300),
		Limitations:   limitations,
	}, nil
}

// RefuseQuestion reports whether the question asks for a side effect or a
// different case. The caller should not send those questions to a model.
func RefuseQuestion(question, caseID string) (bool, string) {
	q := strings.TrimSpace(question)
	if q == "" {
		return true, "Question is empty."
	}
	if sideEffect.MatchString(q) {
		return true, "This assistant cannot freeze accounts, move money, or accuse anyone. Record a human decision in the case workflow if a review action is needed."
	}
	if other := otherCase(q, caseID); other {
		return true, "Questions are limited to the open case. This assistant will not retrieve another case."
	}
	return false, ""
}

func hasToolRequest(raw string) bool {
	var generic map[string]any
	if err := json.Unmarshal([]byte(extractJSON(raw)), &generic); err != nil {
		return false
	}
	for _, key := range []string{"tool_call", "tool_calls", "functionCall", "function_call", "tools"} {
		if _, ok := generic[key]; ok {
			return true
		}
	}
	return false
}

func otherCase(question, caseID string) bool {
	for _, field := range strings.Fields(question) {
		field = strings.Trim(field, ".,;:\"'`")
		if strings.HasPrefix(field, "case_") && field != caseID {
			return true
		}
	}
	return false
}

func cleanHypotheses(in []Hypothesis, allow map[string]struct{}) ([]Hypothesis, []Unverified) {
	var out []Hypothesis
	var bad []Unverified
	for _, h := range in {
		claim := clip(h.Claim, 500)
		if claim == "" {
			continue
		}
		if accusation.MatchString(claim) {
			bad = append(bad, Unverified{Claim: claim, Reason: "accusation language is not a supported hypothesis"})
			continue
		}
		unc := h.Uncertainty
		switch unc {
		case "low", "medium", "high":
		default:
			unc = "high"
		}
		ids := filterIDs(h.EvidenceIDs, allow)
		if len(ids) == 0 {
			unc = "high"
		}
		out = append(out, Hypothesis{Claim: claim, EvidenceIDs: ids, Uncertainty: unc})
	}
	return out, bad
}

func filterIDs(ids []string, allow map[string]struct{}) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, id := range ids {
		if _, ok := allow[id]; !ok {
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

func setOf(ids []string) map[string]struct{} {
	m := map[string]struct{}{}
	for _, id := range ids {
		m[id] = struct{}{}
	}
	return m
}

func extractJSON(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	start := strings.IndexByte(s, '{')
	end := strings.LastIndexByte(s, '}')
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func clipList(in []string, maxItems, maxLen int) []string {
	var out []string
	for _, item := range in {
		item = clip(item, maxLen)
		if item == "" {
			continue
		}
		out = append(out, item)
		if len(out) == maxItems {
			break
		}
	}
	if out == nil {
		return []string{}
	}
	return out
}

func capFacts(in []Fact, n int) []Fact {
	if len(in) > n {
		return in[:n]
	}
	if in == nil {
		return []Fact{}
	}
	return in
}

func capHyp(in []Hypothesis, n int) []Hypothesis {
	if len(in) > n {
		return in[:n]
	}
	if in == nil {
		return []Hypothesis{}
	}
	return in
}

func capUnverified(in []Unverified, n int) []Unverified {
	if len(in) > n {
		return in[:n]
	}
	if in == nil {
		return []Unverified{}
	}
	return in
}

// ErrPoison means the model output asked for a tool or control action.
var ErrPoison = errors.New("model output rejected")
