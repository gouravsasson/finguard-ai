// Package rules evaluates deterministic, versioned risk signals.
// It never calls a model and never assigns guilt.
package rules

import (
	"fmt"
	"sort"
	"time"
)

const Version = "2026.1"

// Config is the versioned rule configuration. Ratios are integers so the same
// inputs always produce the same comparison.
type Config struct {
	VelocityWindow          time.Duration
	VelocityMinCount        int
	AmountRatioNum          int64
	AmountRatioDen          int64
	AmountMinPriors         int
	AmountMinMinor          int64
	FailureWindow           time.Duration
	FailureMinCount         int
	GeoWindow               time.Duration
	SharedDeviceMinAccounts int
	MerchantNoveltyMinMinor int64
}

func DefaultConfig() Config {
	return Config{
		VelocityWindow:          10 * time.Minute,
		VelocityMinCount:        5,
		AmountRatioNum:          3,
		AmountRatioDen:          1,
		AmountMinPriors:         4,
		AmountMinMinor:          10_000,
		FailureWindow:           30 * time.Minute,
		FailureMinCount:         3,
		GeoWindow:               2 * time.Hour,
		SharedDeviceMinAccounts: 2,
		MerchantNoveltyMinMinor: 50_000,
	}
}

// Transaction is the only input the engine may use. Free-text memos are
// intentionally absent so prompt injection in a memo cannot change a signal.
type Transaction struct {
	ID          string
	AccountID   string
	MerchantID  string
	DeviceID    string
	Country     string
	AmountMinor int64
	Status      string
	OccurredAt  time.Time
}

// History is explicit prior activity. DeviceAccountIDs is an observation count,
// not an identity graph.
type History struct {
	Account            []Transaction
	DeviceAccountIDs   []string
	MerchantSeenBefore bool
}

type Signal struct {
	RuleCode      string
	RuleVersion   string
	Explanation   string
	Threshold     string
	Observed      string
	Inputs        map[string]any
	RelatedTxnIDs []string
}

// Evaluate returns signals sorted by rule code. Output is a pure function of
// cfg, current, and hist.
func Evaluate(cfg Config, current Transaction, hist History) []Signal {
	txns := append([]Transaction(nil), hist.Account...)
	sort.Slice(txns, func(i, j int) bool {
		if txns[i].OccurredAt.Equal(txns[j].OccurredAt) {
			return txns[i].ID < txns[j].ID
		}
		return txns[i].OccurredAt.Before(txns[j].OccurredAt)
	})
	var out []Signal
	if s, ok := ruleVelocity(cfg, current, txns); ok {
		out = append(out, s)
	}
	if s, ok := ruleAmount(cfg, current, txns); ok {
		out = append(out, s)
	}
	if s, ok := ruleFailures(cfg, current, txns); ok {
		out = append(out, s)
	}
	if s, ok := ruleSharedDevice(cfg, current, hist.DeviceAccountIDs); ok {
		out = append(out, s)
	}
	if s, ok := ruleGeo(cfg, current, txns); ok {
		out = append(out, s)
	}
	if s, ok := ruleMerchant(cfg, current, hist.MerchantSeenBefore); ok {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RuleCode < out[j].RuleCode })
	return out
}

func countsAsActivity(status string) bool {
	return status == "approved" || status == "declined"
}

func inWindow(current, other time.Time, window time.Duration) bool {
	if other.After(current) {
		return false
	}
	start := current.Add(-window)
	return !other.Before(start)
}

func uniqueSorted(ids []string) []string {
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

func ruleVelocity(cfg Config, current Transaction, txns []Transaction) (Signal, bool) {
	var ids []string
	for _, tx := range txns {
		if tx.AccountID != current.AccountID || !countsAsActivity(tx.Status) {
			continue
		}
		if inWindow(current.OccurredAt, tx.OccurredAt, cfg.VelocityWindow) {
			ids = append(ids, tx.ID)
		}
	}
	ids = uniqueSorted(ids)
	if len(ids) < cfg.VelocityMinCount {
		return Signal{}, false
	}
	return Signal{
		RuleCode:    "velocity",
		RuleVersion: Version,
		Explanation: fmt.Sprintf("%d transactions for this account occurred within %s (threshold %d). This is a velocity observation, not a determination of fraud.", len(ids), cfg.VelocityWindow, cfg.VelocityMinCount),
		Threshold:   fmt.Sprintf("count>=%d within %s", cfg.VelocityMinCount, cfg.VelocityWindow),
		Observed:    fmt.Sprintf("count=%d", len(ids)),
		Inputs: map[string]any{
			"window":    cfg.VelocityWindow.String(),
			"threshold": cfg.VelocityMinCount,
			"observed":  len(ids),
		},
		RelatedTxnIDs: ids,
	}, true
}

func ruleAmount(cfg Config, current Transaction, txns []Transaction) (Signal, bool) {
	if current.Status != "approved" || current.AmountMinor < cfg.AmountMinMinor {
		return Signal{}, false
	}
	var priors []int64
	var ids []string
	for _, tx := range txns {
		if tx.ID == current.ID || tx.AccountID != current.AccountID || tx.Status != "approved" {
			continue
		}
		if !tx.OccurredAt.Before(current.OccurredAt) {
			continue
		}
		priors = append(priors, tx.AmountMinor)
		ids = append(ids, tx.ID)
	}
	if len(priors) < cfg.AmountMinPriors {
		return Signal{}, false
	}
	// Lower-middle median keeps the comparison integer and stable for even counts.
	sorted := append([]int64(nil), priors...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	med := sorted[(len(sorted)-1)/2]
	if med <= 0 || current.AmountMinor*cfg.AmountRatioDen < med*cfg.AmountRatioNum {
		return Signal{}, false
	}
	ids = append(ids, current.ID)
	return Signal{
		RuleCode:    "amount_deviation",
		RuleVersion: Version,
		Explanation: fmt.Sprintf("Approved amount %d minor units is at least %d/%d times the prior lower-middle median %d (minimum %d priors). This compares amounts; it does not establish fraud.", current.AmountMinor, cfg.AmountRatioNum, cfg.AmountRatioDen, med, cfg.AmountMinPriors),
		Threshold:   fmt.Sprintf("amount>=%d/%d*median and priors>=%d and amount>=%d", cfg.AmountRatioNum, cfg.AmountRatioDen, cfg.AmountMinPriors, cfg.AmountMinMinor),
		Observed:    fmt.Sprintf("amount=%d median=%d priors=%d", current.AmountMinor, med, len(priors)),
		Inputs: map[string]any{
			"amount_minor": current.AmountMinor,
			"median_minor": med,
			"priors":       len(priors),
			"ratio_num":    cfg.AmountRatioNum,
			"ratio_den":    cfg.AmountRatioDen,
		},
		RelatedTxnIDs: uniqueSorted(ids),
	}, true
}

func ruleFailures(cfg Config, current Transaction, txns []Transaction) (Signal, bool) {
	if current.Status != "declined" {
		return Signal{}, false
	}
	var ids []string
	for _, tx := range txns {
		if tx.AccountID != current.AccountID || tx.Status != "declined" {
			continue
		}
		if inWindow(current.OccurredAt, tx.OccurredAt, cfg.FailureWindow) {
			ids = append(ids, tx.ID)
		}
	}
	ids = uniqueSorted(ids)
	if len(ids) < cfg.FailureMinCount {
		return Signal{}, false
	}
	return Signal{
		RuleCode:    "repeated_failures",
		RuleVersion: Version,
		Explanation: fmt.Sprintf("%d declined transactions for this account occurred within %s (threshold %d). Repeated declines are a review signal, not proof of takeover.", len(ids), cfg.FailureWindow, cfg.FailureMinCount),
		Threshold:   fmt.Sprintf("declines>=%d within %s", cfg.FailureMinCount, cfg.FailureWindow),
		Observed:    fmt.Sprintf("declines=%d", len(ids)),
		Inputs: map[string]any{
			"window":    cfg.FailureWindow.String(),
			"threshold": cfg.FailureMinCount,
			"observed":  len(ids),
		},
		RelatedTxnIDs: ids,
	}, true
}

func ruleSharedDevice(cfg Config, current Transaction, accountIDs []string) (Signal, bool) {
	if current.DeviceID == "" {
		return Signal{}, false
	}
	ids := uniqueSorted(accountIDs)
	if len(ids) < cfg.SharedDeviceMinAccounts {
		return Signal{}, false
	}
	return Signal{
		RuleCode:    "shared_device",
		RuleVersion: Version,
		Explanation: fmt.Sprintf("Device %s was observed on %d accounts in this tenant (threshold %d). A shared device is not evidence that the accounts belong to the same person.", current.DeviceID, len(ids), cfg.SharedDeviceMinAccounts),
		Threshold:   fmt.Sprintf("distinct_accounts>=%d", cfg.SharedDeviceMinAccounts),
		Observed:    fmt.Sprintf("distinct_accounts=%d", len(ids)),
		Inputs: map[string]any{
			"device_id":          current.DeviceID,
			"distinct_accounts":  len(ids),
			"threshold":          cfg.SharedDeviceMinAccounts,
			"identity_inference": false,
		},
		RelatedTxnIDs: []string{current.ID},
	}, true
}

func ruleGeo(cfg Config, current Transaction, txns []Transaction) (Signal, bool) {
	cur := upper(current.Country)
	if len(cur) != 2 {
		return Signal{}, false
	}
	var prior Transaction
	found := false
	for _, tx := range txns {
		if tx.ID == current.ID || tx.AccountID != current.AccountID {
			continue
		}
		if !inWindow(current.OccurredAt, tx.OccurredAt, cfg.GeoWindow) {
			continue
		}
		country := upper(tx.Country)
		if len(country) != 2 || country == cur {
			continue
		}
		if !found || tx.OccurredAt.After(prior.OccurredAt) {
			prior = tx
			found = true
		}
	}
	if !found {
		return Signal{}, false
	}
	return Signal{
		RuleCode:    "geographic_inconsistency",
		RuleVersion: Version,
		Explanation: fmt.Sprintf("This transaction is in %s and a prior transaction %s is in %s, within %s. Country change is context for review, not proof of compromise. Travel and network location error remain possible.", cur, prior.ID, upper(prior.Country), cfg.GeoWindow),
		Threshold:   fmt.Sprintf("country differs within %s", cfg.GeoWindow),
		Observed:    fmt.Sprintf("%s->%s", upper(prior.Country), cur),
		Inputs: map[string]any{
			"current_country": cur,
			"prior_country":   upper(prior.Country),
			"prior_txn_id":    prior.ID,
			"window":          cfg.GeoWindow.String(),
		},
		RelatedTxnIDs: uniqueSorted([]string{current.ID, prior.ID}),
	}, true
}

func ruleMerchant(cfg Config, current Transaction, seen bool) (Signal, bool) {
	if seen || current.MerchantID == "" || current.Status != "approved" || current.AmountMinor < cfg.MerchantNoveltyMinMinor {
		return Signal{}, false
	}
	return Signal{
		RuleCode:    "merchant_novelty",
		RuleVersion: Version,
		Explanation: fmt.Sprintf("This is the first observed approved transaction at merchant %s and the amount is %d minor units (threshold %d). A new merchant is not by itself suspicious.", current.MerchantID, current.AmountMinor, cfg.MerchantNoveltyMinMinor),
		Threshold:   fmt.Sprintf("first_merchant and amount>=%d", cfg.MerchantNoveltyMinMinor),
		Observed:    fmt.Sprintf("amount=%d first_seen=true", current.AmountMinor),
		Inputs: map[string]any{
			"merchant_id":  current.MerchantID,
			"amount_minor": current.AmountMinor,
			"first_seen":   true,
		},
		RelatedTxnIDs: []string{current.ID},
	}, true
}

func upper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
	}
	return string(b)
}
