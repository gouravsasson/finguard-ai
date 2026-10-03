package aigateway

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

// Stub writes a grounded report from the evidence ids supplied to it.
// It is a local fallback. Callers must label provider=local-stub and live=false.
type Stub struct{}

func (Stub) Generate(_ context.Context, req Request) (Response, error) {
	start := time.Now()
	if req.PromptVersion == AssistantPromptVersion {
		return assistantStub(req, start), nil
	}
	if req.PromptVersion == DissentPromptVersion {
		return dissentStub(req, start), nil
	}
	ids := req.EvidenceIDs
	var facts []map[string]any
	var steps []string
	steps = append(steps,
		"Read each cited evidence record beside the claim.",
		"Compare the deterministic rule inputs with the source transactions.",
		"Record a human rationale before any case transition. Do not freeze an account or move money from this tool.",
	)
	summary := "No deterministic signals were attached to the evidence package supplied for this case."
	if len(ids) > 0 {
		summary = "The evidence package contains deterministic review signals. Each statement below cites only identifiers present in that package. This is not a finding that anyone committed fraud."
		limit := len(ids)
		if limit > 6 {
			limit = 6
		}
		for _, id := range ids[:limit] {
			facts = append(facts, map[string]any{
				"claim":        "Evidence " + id + " is part of the package assembled for this case and should be read before a human decision.",
				"evidence_ids": []string{id},
			})
		}
	}
	alts := []map[string]any{
		{
			"claim":        "The activity may be legitimate, including a new device, household use, travel, or a merchant retry. The package does not rule those explanations in or out.",
			"evidence_ids": nonEmptyFirst(ids),
			"uncertainty":  "high",
		},
	}
	missing := []string{
		"No customer contact result is in the package.",
		"No authentication or session event is in the package.",
		"No independent confirmation that a device belongs to the account holder.",
	}
	limitations := []string{
		"This report was produced by the local stub because a live model call was not used.",
		"The stub restates supplied evidence identifiers. It does not add facts.",
		"Shared device or unusual velocity is not an identity or guilt conclusion.",
	}
	if strings.Contains(req.User, `"truncated":true`) || strings.Contains(req.User, `"truncated": true`) {
		limitations = append(limitations, "The evidence package was truncated to a bounded size before this report was drafted.")
	}
	body := map[string]any{
		"summary":        summary,
		"observed_facts": facts,
		"hypotheses": []map[string]any{
			{
				"claim":        "A human investigator may want to review the cited signals together. The package does not establish what caused them.",
				"evidence_ids": nonEmptyFirst(ids),
				"uncertainty":  "high",
			},
		},
		"alternative_explanations": alts,
		"missing_evidence":         missing,
		"suggested_review_steps":   steps,
		"limitations":              limitations,
		"model_metadata": map[string]any{
			"note": "server replaces provider metadata",
		},
	}
	raw, _ := json.Marshal(body)
	return Response{
		Text:     string(raw),
		Model:    "finguard-stub-2026.1",
		Provider: "local-stub",
		Live:     false,
		Latency:  time.Since(start),
	}, nil
}

func assistantStub(req Request, start time.Time) Response {
	ids := req.EvidenceIDs
	answer := "The supplied evidence does not contain enough information to answer that from this case alone."
	var cites []string
	if len(ids) > 0 {
		cites = []string{ids[0]}
		answer = "From the evidence available in this case, the supported statement is limited to what " + ids[0] + " and the attached rule explanations record. I cannot infer intent or identity beyond that."
	}
	body := map[string]any{
		"answer":         answer,
		"citations":      cites,
		"uncertainty":    "high",
		"refused":        false,
		"refusal_reason": "",
		"limitations":    []string{"Local stub answer. No live model was called.", "Only evidence ids from this case were available."},
	}
	raw, _ := json.Marshal(body)
	return Response{Text: string(raw), Model: "finguard-stub-2026.1", Provider: "local-stub", Live: false, Latency: time.Since(start)}
}

func dissentStub(req Request, start time.Time) Response {
	body := map[string]any{
		"concern":           map[string]any{"claim": "The replay data did not include a reproducible signal to narrate.", "evidence_ids": nonEmptyFirst(req.EvidenceIDs)},
		"benign":            map[string]any{"claim": "No clearing replay was available in the supplied data.", "evidence_ids": nonEmptyFirst(req.EvidenceIDs)},
		"what_would_change": []any{},
		"limitations":       []string{"This dissent was written by the local stub. No live model was called.", "The stub only restates replay rows supplied to it."},
	}
	startIdx := strings.Index(req.User, "REPLAY_BEGIN")
	endIdx := strings.Index(req.User, "REPLAY_END")
	if startIdx >= 0 && endIdx > startIdx {
		raw := strings.TrimSpace(req.User[startIdx+len("REPLAY_BEGIN") : endIdx])
		var board struct {
			Signals []struct {
				Explanation string `json:"explanation"`
				Reproduced  bool   `json:"reproduced"`
				Attempts    []struct {
					Title       string   `json:"title"`
					Detail      string   `json:"detail"`
					Clears      bool     `json:"clears"`
					EvidenceIDs []string `json:"evidence_ids"`
				} `json:"attempts"`
			} `json:"signals"`
		}
		if json.Unmarshal([]byte(raw), &board) == nil {
			var changes []any
			for _, sig := range board.Signals {
				if !sig.Reproduced || sig.Explanation == "" {
					continue
				}
				cite := nonEmptyFirst(req.EvidenceIDs)
				if body["concern"].(map[string]any)["claim"] == "The replay data did not include a reproducible signal to narrate." {
					if len(sig.Attempts) > 0 && len(sig.Attempts[0].EvidenceIDs) > 0 {
						cite = sig.Attempts[0].EvidenceIDs
					}
					body["concern"] = map[string]any{"claim": sig.Explanation, "evidence_ids": cite}
				}
				for _, attempt := range sig.Attempts {
					if !attempt.Clears || attempt.Detail == "" {
						continue
					}
					ids := attempt.EvidenceIDs
					if len(ids) == 0 {
						ids = cite
					}
					if body["benign"].(map[string]any)["claim"] == "No clearing replay was available in the supplied data." {
						body["benign"] = map[string]any{"claim": attempt.Detail, "evidence_ids": ids}
					}
					if len(changes) < 3 {
						changes = append(changes, map[string]any{"claim": attempt.Title + ": " + attempt.Detail, "evidence_ids": ids})
					}
				}
			}
			body["what_would_change"] = changes
		}
	}
	raw, _ := json.Marshal(body)
	return Response{Text: string(raw), Model: "finguard-stub-2026.1", Provider: "local-stub", Live: false, Latency: time.Since(start)}
}

func nonEmptyFirst(ids []string) []string {
	if len(ids) == 0 {
		return []string{}
	}
	return []string{ids[0]}
}
