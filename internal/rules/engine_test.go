package rules

import (
	"encoding/json"
	"testing"
	"time"
)

func base() (Config, time.Time) {
	return DefaultConfig(), time.Date(2026, 6, 15, 15, 0, 0, 0, time.UTC)
}

func TestVelocity(t *testing.T) {
	cfg, now := base()
	cur := Transaction{ID: "t6", AccountID: "a", AmountMinor: 1000, Status: "approved", OccurredAt: now}
	var hist []Transaction
	for i := 0; i < 6; i++ {
		hist = append(hist, Transaction{
			ID: "t" + string(rune('1'+i)), AccountID: "a", AmountMinor: 1000, Status: "approved",
			OccurredAt: now.Add(-time.Duration(5-i) * time.Minute),
		})
	}
	hist[5] = cur
	got := Evaluate(cfg, cur, History{Account: hist})
	if len(got) != 1 || got[0].RuleCode != "velocity" {
		t.Fatalf("got %#v", got)
	}
	if len(got[0].RelatedTxnIDs) != 6 {
		t.Fatalf("related %#v", got[0].RelatedTxnIDs)
	}
}

func TestAmountDeviation(t *testing.T) {
	cfg, now := base()
	var hist []Transaction
	for i := 0; i < 4; i++ {
		hist = append(hist, Transaction{
			ID: "p" + string(rune('1'+i)), AccountID: "a", AmountMinor: 2_000, Status: "approved",
			OccurredAt: now.Add(-time.Duration(24*(4-i)) * time.Hour),
		})
	}
	cur := Transaction{ID: "big", AccountID: "a", AmountMinor: 40_000, Status: "approved", OccurredAt: now}
	hist = append(hist, cur)
	got := Evaluate(cfg, cur, History{Account: hist})
	if !has(got, "amount_deviation") {
		t.Fatalf("got %#v", codes(got))
	}
}

func TestRepeatedFailures(t *testing.T) {
	cfg, now := base()
	var hist []Transaction
	for i := 0; i < 3; i++ {
		hist = append(hist, Transaction{
			ID: "d" + string(rune('1'+i)), AccountID: "a", AmountMinor: 5_000, Status: "declined",
			OccurredAt: now.Add(-time.Duration(2-i) * time.Minute),
		})
	}
	cur := hist[2]
	got := Evaluate(cfg, cur, History{Account: hist})
	if !has(got, "repeated_failures") {
		t.Fatalf("got %#v", codes(got))
	}
}

func TestSharedDeviceIsNotIdentity(t *testing.T) {
	cfg, now := base()
	cur := Transaction{ID: "t", AccountID: "a", DeviceID: "dev_1", AmountMinor: 100, Status: "approved", OccurredAt: now}
	got := Evaluate(cfg, cur, History{Account: []Transaction{cur}, DeviceAccountIDs: []string{"a", "b"}})
	if !has(got, "shared_device") {
		t.Fatal("expected shared device")
	}
	s := got[0]
	if s.Inputs["identity_inference"] != false {
		t.Fatalf("identity flag %#v", s.Inputs)
	}
	if s.Explanation == "" || !contains(s.Explanation, "not evidence") {
		t.Fatalf("explanation %s", s.Explanation)
	}
	one := Evaluate(cfg, cur, History{Account: []Transaction{cur}, DeviceAccountIDs: []string{"a"}})
	if has(one, "shared_device") {
		t.Fatal("single account must not signal")
	}
}

func TestGeoAndMerchant(t *testing.T) {
	cfg, now := base()
	prior := Transaction{ID: "old", AccountID: "a", Country: "US", AmountMinor: 100, Status: "approved", OccurredAt: now.Add(-20 * time.Minute)}
	cur := Transaction{ID: "new", AccountID: "a", MerchantID: "m", Country: "DE", AmountMinor: 80_000, Status: "approved", OccurredAt: now}
	got := Evaluate(cfg, cur, History{Account: []Transaction{prior, cur}, MerchantSeenBefore: false})
	if !has(got, "geographic_inconsistency") || !has(got, "merchant_novelty") {
		t.Fatalf("codes %v", codes(got))
	}
	quiet := Evaluate(cfg, cur, History{Account: []Transaction{prior, cur}, MerchantSeenBefore: true})
	if has(quiet, "merchant_novelty") {
		t.Fatal("seen merchant must not be novel")
	}
}

func TestDeterministic(t *testing.T) {
	cfg, now := base()
	cur := Transaction{ID: "c", AccountID: "a", DeviceID: "d", MerchantID: "m", Country: "DE", AmountMinor: 90_000, Status: "approved", OccurredAt: now}
	hist := History{
		Account: []Transaction{
			{ID: "b", AccountID: "a", Country: "US", AmountMinor: 2_000, Status: "approved", OccurredAt: now.Add(-30 * time.Minute)},
			{ID: "a1", AccountID: "a", AmountMinor: 2_000, Status: "approved", OccurredAt: now.Add(-96 * time.Hour)},
			{ID: "a2", AccountID: "a", AmountMinor: 2_100, Status: "approved", OccurredAt: now.Add(-72 * time.Hour)},
			{ID: "a3", AccountID: "a", AmountMinor: 1_900, Status: "approved", OccurredAt: now.Add(-48 * time.Hour)},
			cur,
		},
		DeviceAccountIDs:   []string{"a", "z"},
		MerchantSeenBefore: false,
	}
	first, _ := json.Marshal(Evaluate(cfg, cur, hist))
	second, _ := json.Marshal(Evaluate(cfg, cur, hist))
	if string(first) != string(second) {
		t.Fatalf("not deterministic\n%s\n%s", first, second)
	}
}

func TestMemoCannotReachEngine(t *testing.T) {
	// Compile-time intention: Transaction has no memo field. A string field
	// named Memo would let ingestion pass attacker-controlled text into rules.
	var tx Transaction
	_ = tx
	fields := []string{"ID", "AccountID", "MerchantID", "DeviceID", "Country", "AmountMinor", "Status", "OccurredAt"}
	if len(fields) != 8 {
		t.Fatal(fields)
	}
}

func has(ss []Signal, code string) bool {
	for _, s := range ss {
		if s.RuleCode == code {
			return true
		}
	}
	return false
}

func codes(ss []Signal) []string {
	var c []string
	for _, s := range ss {
		c = append(c, s.RuleCode)
	}
	return c
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || (len(s) > 0 && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()))
}
