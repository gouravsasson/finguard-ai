// Package knockout replays deterministic rules under hypothetical edits.
// It never writes transactions and never decides that anyone committed fraud.
package knockout

import (
	"fmt"
	"sort"

	"finguard.ai/internal/rules"
)

// Attempt is one hypothetical replay. Clears means the named rule no longer
// matches after the edit. The edit is not applied to stored transactions.
type Attempt struct {
	Kind           string   `json:"kind"`
	Title          string   `json:"title"`
	Detail         string   `json:"detail"`
	Clears         bool     `json:"clears"`
	TransactionIDs []string `json:"transaction_ids"`
	EvidenceIDs    []string `json:"evidence_ids,omitempty"`
}

// SignalReplay is the knockout desk for one stored signal.
type SignalReplay struct {
	SignalID       string    `json:"signal_id"`
	RuleCode       string    `json:"rule_code"`
	TransactionID  string    `json:"transaction_id"`
	Explanation    string    `json:"explanation"`
	Reproduced     bool      `json:"reproduced"`
	ReplayObserved string    `json:"replay_observed,omitempty"`
	Note           string    `json:"note,omitempty"`
	Attempts       []Attempt `json:"attempts"`
}

// Board is the case-level knockout desk.
type Board struct {
	Method      string         `json:"method"`
	RuleVersion string         `json:"rule_version"`
	Limitation  string         `json:"limitation"`
	Signals     []SignalReplay `json:"signals"`
	Reading     Reading        `json:"reading"`
}

// Reading is a deterministic dissent written only from replay results.
type Reading struct {
	Concern         string   `json:"concern"`
	ConcernEvidence []string `json:"concern_evidence"`
	Benign          string   `json:"benign"`
	BenignEvidence  []string `json:"benign_evidence"`
	WouldChange     []string `json:"would_change"`
	ChangeEvidence  []string `json:"change_evidence"`
}

// Replay asks whether small hypothetical edits remove one rule. The returned
// attempts are ordered and capped so the desk stays readable.
func Replay(cfg rules.Config, ruleCode string, trigger rules.Transaction, hist rules.History) SignalReplay {
	out := SignalReplay{RuleCode: ruleCode, TransactionID: trigger.ID, Attempts: []Attempt{}}
	baseline, ok := findRule(cfg, trigger, hist, ruleCode)
	if !ok {
		out.Note = "This stored signal did not reproduce from the account history available for replay. The stored explanation is unchanged."
		return out
	}
	out.Reproduced = true
	out.Explanation = baseline.Explanation
	out.ReplayObserved = baseline.Observed
	out.Attempts = attemptsFor(cfg, baseline, trigger, hist)
	return out
}

func attemptsFor(cfg rules.Config, baseline rules.Signal, trigger rules.Transaction, hist rules.History) []Attempt {
	var attempts []Attempt
	related := append([]string(nil), baseline.RelatedTxnIDs...)
	sort.Strings(related)
	if len(related) > 6 {
		related = related[:6]
	}
	for _, id := range related {
		attempts = append(attempts, holdOut(cfg, baseline, trigger, hist, []string{id}))
	}
	switch baseline.RuleCode {
	case "velocity", "repeated_failures":
		if extra := smallestCountDrop(cfg, baseline, trigger, hist); extra.Kind != "" {
			attempts = append(attempts, extra)
		}
	case "amount_deviation":
		attempts = append(attempts, amountDrop(cfg, baseline, trigger, hist))
	case "geographic_inconsistency":
		attempts = append(attempts, sameCountry(cfg, baseline, trigger, hist))
	case "shared_device":
		attempts = append(attempts, sharedControl(cfg, baseline, trigger, hist))
		attempts = append(attempts, singleAccountDevice(cfg, baseline, trigger, hist))
	case "merchant_novelty":
		attempts = append(attempts, merchantSeen(cfg, baseline, trigger, hist))
		attempts = append(attempts, merchantAmount(cfg, trigger, hist))
	}
	if len(attempts) > 8 {
		attempts = attempts[:8]
	}
	return attempts
}

func holdOut(cfg rules.Config, baseline rules.Signal, trigger rules.Transaction, hist rules.History, drop []string) Attempt {
	next, current, ok := applyDrop(trigger, hist, drop)
	attempt := Attempt{
		Kind:           "hold_out",
		Title:          "Hold out " + dropLabel(drop),
		TransactionIDs: append([]string(nil), drop...),
	}
	if !ok {
		attempt.Clears = true
		attempt.Detail = "Holding out " + dropLabel(drop) + " leaves no remaining transaction on which this rule still matches."
		return attempt
	}
	still, ok := findRule(cfg, current, next, baseline.RuleCode)
	if ok {
		attempt.Clears = false
		attempt.Detail = fmt.Sprintf("Holding out %s still leaves %s in place (%s). The remaining records still meet the threshold.", dropLabel(drop), baseline.RuleCode, still.Observed)
		return attempt
	}
	attempt.Clears = true
	attempt.Detail = fmt.Sprintf("Holding out %s drops %s below its threshold. The stored observation was %s.", dropLabel(drop), baseline.RuleCode, baseline.Observed)
	return attempt
}

func smallestCountDrop(cfg rules.Config, baseline rules.Signal, trigger rules.Transaction, hist rules.History) Attempt {
	observed := asInt(baseline.Inputs["observed"])
	threshold := cfg.VelocityMinCount
	if baseline.RuleCode == "repeated_failures" {
		threshold = cfg.FailureMinCount
	}
	need := observed - threshold + 1
	if need <= 1 {
		return Attempt{}
	}
	var drop []string
	for _, id := range baseline.RelatedTxnIDs {
		if id == trigger.ID {
			continue
		}
		drop = append(drop, id)
		if len(drop) == need {
			break
		}
	}
	if len(drop) < need {
		return Attempt{}
	}
	sort.Strings(drop)
	attempt := holdOut(cfg, baseline, trigger, hist, drop)
	attempt.Kind = "smallest_count_drop"
	attempt.Title = fmt.Sprintf("Hold out %d transactions in the window", len(drop))
	if attempt.Clears {
		attempt.Detail = fmt.Sprintf("The replay count is %d against a threshold of %d. Holding out these %d transactions is the smallest set, among those tried, that drops the rule. Holding out only one does not.", observed, threshold, len(drop))
	}
	return attempt
}

func amountDrop(cfg rules.Config, baseline rules.Signal, trigger rules.Transaction, hist rules.History) Attempt {
	med := asInt64(baseline.Inputs["median_minor"])
	next := trigger
	if med > 0 {
		next.AmountMinor = med*cfg.AmountRatioNum/cfg.AmountRatioDen - 1
	}
	if next.AmountMinor < 0 {
		next.AmountMinor = 0
	}
	still, ok := findRule(cfg, next, replaceTrigger(hist, next), "amount_deviation")
	attempt := Attempt{
		Kind:           "reduce_amount",
		Title:          "Replay the amount just below the ratio",
		TransactionIDs: []string{trigger.ID},
		Clears:         !ok,
	}
	if attempt.Clears {
		attempt.Detail = fmt.Sprintf("Setting this amount to %d minor units, just under %d/%d of the prior lower-middle median, drops amount deviation. The stored observation was %s.", next.AmountMinor, cfg.AmountRatioNum, cfg.AmountRatioDen, baseline.Observed)
	} else {
		attempt.Detail = fmt.Sprintf("Reducing the amount to %d minor units still leaves amount deviation (%s).", next.AmountMinor, still.Observed)
	}
	return attempt
}

func sameCountry(cfg rules.Config, baseline rules.Signal, trigger rules.Transaction, hist rules.History) Attempt {
	prior, _ := baseline.Inputs["prior_country"].(string)
	next := trigger
	next.Country = prior
	still, ok := findRule(cfg, next, replaceTrigger(hist, next), "geographic_inconsistency")
	attempt := Attempt{
		Kind:           "same_country",
		Title:          "Replay this transaction in the prior country",
		TransactionIDs: []string{trigger.ID},
		Clears:         !ok,
	}
	if attempt.Clears {
		attempt.Detail = fmt.Sprintf("If this transaction were in %s, the same country as the prior transaction inside %s, geographic inconsistency would not match. Travel and a wrong location remain possible either way.", prior, cfg.GeoWindow)
	} else {
		attempt.Detail = fmt.Sprintf("Replaying the country as %s still leaves geographic inconsistency (%s).", prior, still.Observed)
	}
	return attempt
}

func sharedControl(cfg rules.Config, baseline rules.Signal, trigger rules.Transaction, hist rules.History) Attempt {
	still, _ := findRule(cfg, trigger, hist, "shared_device")
	n := asInt(baseline.Inputs["distinct_accounts"])
	return Attempt{
		Kind:           "keep_observed_accounts",
		Title:          "Leave the observed account set unchanged",
		TransactionIDs: []string{trigger.ID},
		Clears:         false,
		Detail:         fmt.Sprintf("The device stays observed on %d accounts (%s). That meets the review threshold. It is still not evidence that the accounts belong to the same person.", n, still.Observed),
	}
}

func singleAccountDevice(cfg rules.Config, baseline rules.Signal, trigger rules.Transaction, hist rules.History) Attempt {
	next := cloneHist(hist)
	if trigger.AccountID != "" {
		next.DeviceAccountIDs = []string{trigger.AccountID}
	} else {
		next.DeviceAccountIDs = nil
	}
	still, ok := findRule(cfg, trigger, next, "shared_device")
	attempt := Attempt{
		Kind:           "single_account_device",
		Title:          "Replay the device on this account only",
		TransactionIDs: []string{trigger.ID},
		Clears:         !ok,
	}
	if attempt.Clears {
		attempt.Detail = "If the device had been observed only on this account, the shared-device rule would not match. That hypothetical does not identify who used it."
	} else {
		attempt.Detail = "Limiting the device to this account still leaves the shared-device rule (" + still.Observed + ")."
	}
	return attempt
}

func merchantSeen(cfg rules.Config, baseline rules.Signal, trigger rules.Transaction, hist rules.History) Attempt {
	next := cloneHist(hist)
	next.MerchantSeenBefore = true
	still, ok := findRule(cfg, trigger, next, "merchant_novelty")
	attempt := Attempt{
		Kind:           "merchant_seen_before",
		Title:          "Replay as a merchant this account has used",
		TransactionIDs: []string{trigger.ID},
		Clears:         !ok,
	}
	if attempt.Clears {
		attempt.Detail = "If an earlier approved transaction at this merchant were already on the account, merchant novelty would not match. The stored observation was " + baseline.Observed + "."
	} else {
		attempt.Detail = "Marking the merchant as previously seen still leaves merchant novelty (" + still.Observed + ")."
	}
	return attempt
}

func merchantAmount(cfg rules.Config, trigger rules.Transaction, hist rules.History) Attempt {
	next := trigger
	next.AmountMinor = cfg.MerchantNoveltyMinMinor - 1
	if next.AmountMinor < 0 {
		next.AmountMinor = 0
	}
	still, ok := findRule(cfg, next, replaceTrigger(hist, next), "merchant_novelty")
	attempt := Attempt{
		Kind:           "merchant_amount_below",
		Title:          "Replay the amount under the novelty floor",
		TransactionIDs: []string{trigger.ID},
		Clears:         !ok,
	}
	if attempt.Clears {
		attempt.Detail = fmt.Sprintf("An amount of %d minor units is under the novelty floor of %d, so the rule would not match even on a first visit to the merchant.", next.AmountMinor, cfg.MerchantNoveltyMinMinor)
	} else {
		attempt.Detail = "Lowering the amount still leaves merchant novelty (" + still.Observed + ")."
	}
	return attempt
}

func applyDrop(trigger rules.Transaction, hist rules.History, drop []string) (rules.History, rules.Transaction, bool) {
	skip := map[string]struct{}{}
	for _, id := range drop {
		skip[id] = struct{}{}
	}
	next := cloneHist(hist)
	var acc []rules.Transaction
	for _, tx := range next.Account {
		if _, ok := skip[tx.ID]; ok {
			continue
		}
		acc = append(acc, tx)
	}
	next.Account = acc
	if _, dropped := skip[trigger.ID]; !dropped {
		return next, trigger, true
	}
	current, ok := latest(acc, trigger.AccountID)
	return next, current, ok
}

func replaceTrigger(hist rules.History, trigger rules.Transaction) rules.History {
	next := cloneHist(hist)
	for i, tx := range next.Account {
		if tx.ID == trigger.ID {
			next.Account[i] = trigger
		}
	}
	return next
}

func findRule(cfg rules.Config, current rules.Transaction, hist rules.History, code string) (rules.Signal, bool) {
	for _, sig := range rules.Evaluate(cfg, current, hist) {
		if sig.RuleCode == code {
			return sig, true
		}
	}
	return rules.Signal{}, false
}

func cloneHist(h rules.History) rules.History {
	h.Account = append([]rules.Transaction(nil), h.Account...)
	h.DeviceAccountIDs = append([]string(nil), h.DeviceAccountIDs...)
	return h
}

func latest(txns []rules.Transaction, account string) (rules.Transaction, bool) {
	var best rules.Transaction
	found := false
	for _, tx := range txns {
		if tx.AccountID != account {
			continue
		}
		if !found || tx.OccurredAt.After(best.OccurredAt) || (tx.OccurredAt.Equal(best.OccurredAt) && tx.ID > best.ID) {
			best = tx
			found = true
		}
	}
	return best, found
}

func dropLabel(ids []string) string {
	if len(ids) == 0 {
		return "nothing"
	}
	if len(ids) == 1 {
		return ids[0]
	}
	return fmt.Sprintf("%d transactions", len(ids))
}

func asInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

func asInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}

// ComposeReading writes the dissent from replay rows only.
func ComposeReading(signals []SignalReplay) Reading {
	reading := Reading{}
	var changes []string
	seenChange := map[string]struct{}{}
	for _, sig := range signals {
		if !sig.Reproduced {
			continue
		}
		if reading.Concern == "" {
			reading.Concern = sig.Explanation
			reading.ConcernEvidence = evidenceOf(sig, false)
		}
		for _, attempt := range sig.Attempts {
			if !attempt.Clears || attempt.Detail == "" {
				continue
			}
			if reading.Benign == "" {
				reading.Benign = attempt.Detail
				reading.BenignEvidence = firstIDs(attempt.EvidenceIDs, reading.ConcernEvidence)
			}
			if _, ok := seenChange[attempt.Detail]; ok {
				continue
			}
			seenChange[attempt.Detail] = struct{}{}
			changes = append(changes, attempt.Title+": "+attempt.Detail)
			reading.ChangeEvidence = append(reading.ChangeEvidence, attempt.EvidenceIDs...)
			if len(changes) == 4 {
				break
			}
		}
		if len(changes) == 4 {
			break
		}
	}
	if reading.Concern == "" {
		reading.Concern = "No stored signal reproduced from the account history, so there is no replayed concern to read."
	}
	if reading.Benign == "" {
		reading.Benign = "None of the hypothetical replays dropped a reproduced rule. That means these small edits would not explain the signals away. It is still not a finding that anyone committed fraud."
	}
	reading.WouldChange = changes
	reading.ConcernEvidence = unique(reading.ConcernEvidence)
	reading.BenignEvidence = unique(reading.BenignEvidence)
	reading.ChangeEvidence = unique(reading.ChangeEvidence)
	return reading
}

func evidenceOf(sig SignalReplay, onlyClearing bool) []string {
	var ids []string
	for _, attempt := range sig.Attempts {
		if onlyClearing && !attempt.Clears {
			continue
		}
		ids = append(ids, attempt.EvidenceIDs...)
	}
	return unique(ids)
}

func firstIDs(primary, fallback []string) []string {
	if len(primary) > 0 {
		return unique(primary)
	}
	return unique(fallback)
}

func unique(ids []string) []string {
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
	sort.Strings(out)
	return out
}
