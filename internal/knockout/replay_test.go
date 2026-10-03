package knockout

import (
	"testing"
	"time"

	"finguard.ai/internal/rules"
)

func TestVelocityHoldOutIsHonest(t *testing.T) {
	cfg := rules.DefaultConfig()
	base := time.Date(2026, 6, 15, 17, 0, 0, 0, time.UTC)
	var hist rules.History
	var trigger rules.Transaction
	for i := 0; i < 6; i++ {
		tx := rules.Transaction{
			ID: "txn_" + string(rune('a'+i)), AccountID: "acct", MerchantID: "cafe", DeviceID: "dev",
			Country: "US", AmountMinor: 1800, Status: "approved", OccurredAt: base.Add(time.Duration(i) * time.Minute),
		}
		hist.Account = append(hist.Account, tx)
		trigger = tx
	}
	replay := Replay(cfg, "velocity", trigger, hist)
	if !replay.Reproduced {
		t.Fatal("velocity did not reproduce")
	}
	var singleClears, bundleClears int
	for _, attempt := range replay.Attempts {
		if attempt.Kind == "hold_out" && len(attempt.TransactionIDs) == 1 && attempt.Clears {
			singleClears++
		}
		if attempt.Kind == "smallest_count_drop" {
			if !attempt.Clears || len(attempt.TransactionIDs) != 2 {
				t.Fatalf("bundle %#v", attempt)
			}
			bundleClears++
		}
	}
	if singleClears != 0 || bundleClears != 1 {
		t.Fatalf("single clears %d bundle %d attempts %#v", singleClears, bundleClears, replay.Attempts)
	}
}

func TestSharedDeviceAndGeoAndMerchant(t *testing.T) {
	cfg := rules.DefaultConfig()
	when := time.Date(2026, 6, 15, 18, 0, 0, 0, time.UTC)
	prior := rules.Transaction{ID: "txn_home", AccountID: "acct", Country: "US", Status: "approved", AmountMinor: 1000, OccurredAt: when.Add(-30 * time.Minute), MerchantID: "old"}
	trigger := rules.Transaction{ID: "txn_away", AccountID: "acct", Country: "DE", Status: "approved", AmountMinor: 80_000, OccurredAt: when, MerchantID: "new", DeviceID: "shared"}
	hist := rules.History{Account: []rules.Transaction{prior, trigger}, DeviceAccountIDs: []string{"acct", "other"}}

	geo := Replay(cfg, "geographic_inconsistency", trigger, hist)
	if !hasClear(geo, "same_country") {
		t.Fatalf("geo %#v", geo.Attempts)
	}
	shared := Replay(cfg, "shared_device", trigger, hist)
	if hasClear(shared, "keep_observed_accounts") || !hasClear(shared, "single_account_device") {
		t.Fatalf("shared %#v", shared.Attempts)
	}
	merchant := Replay(cfg, "merchant_novelty", trigger, hist)
	if !hasClear(merchant, "merchant_seen_before") || !hasClear(merchant, "merchant_amount_below") {
		t.Fatalf("merchant %#v", merchant.Attempts)
	}
}

func TestAmountDropClears(t *testing.T) {
	cfg := rules.DefaultConfig()
	base := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	var hist rules.History
	for i := 0; i < 4; i++ {
		hist.Account = append(hist.Account, rules.Transaction{
			ID: "prior_" + string(rune('a'+i)), AccountID: "acct", Status: "approved", AmountMinor: 2000,
			OccurredAt: base.Add(time.Duration(i) * time.Hour), Country: "US",
		})
	}
	trigger := rules.Transaction{ID: "big", AccountID: "acct", Status: "approved", AmountMinor: 20_000, OccurredAt: base.Add(8 * time.Hour), Country: "US"}
	hist.Account = append(hist.Account, trigger)
	replay := Replay(cfg, "amount_deviation", trigger, hist)
	if !replay.Reproduced || !hasClear(replay, "reduce_amount") {
		t.Fatalf("%#v", replay)
	}
}

func hasClear(replay SignalReplay, kind string) bool {
	for _, attempt := range replay.Attempts {
		if attempt.Kind == kind && attempt.Clears && attempt.Detail != "" {
			return true
		}
	}
	return false
}
