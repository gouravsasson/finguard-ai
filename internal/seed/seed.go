// Package seed loads synthetic tenants, people, and activity.
// Ground-truth labels live in the scenario catalog and are not sent to a model.
package seed

import (
	"context"
	"time"

	"finguard.ai/internal/auth"
	"finguard.ai/internal/domain"
	"finguard.ai/internal/ingestion"
	"finguard.ai/internal/rules"
	"finguard.ai/internal/store"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	TenantNorthwind = "ten_northwind"
	TenantSouthbank = "ten_southbank"
)

type Scenario struct {
	ID            string   `json:"id"`
	Account       string   `json:"account"`
	ExpectedRules []string `json:"expected_rules"`
	Legitimate    bool     `json:"legitimate"`
	Narrative     string   `json:"narrative"`
	OpensCase     bool     `json:"opens_case"`
}

func Catalog() []Scenario {
	return []Scenario{
		{ID: "normal_activity", Account: "acct_ava", ExpectedRules: []string{}, Legitimate: true, OpensCase: false, Narrative: "Ordinary purchases spread across days."},
		{ID: "velocity_burst", Account: "acct_milo", ExpectedRules: []string{"velocity"}, OpensCase: true, Narrative: "Six purchases in a few minutes."},
		{ID: "amount_deviation", Account: "acct_lee", ExpectedRules: []string{"amount_deviation"}, OpensCase: true, Narrative: "One large purchase against a small approved baseline."},
		{ID: "repeated_failures", Account: "acct_noor", ExpectedRules: []string{"repeated_failures"}, OpensCase: true, Narrative: "Three declines in a short window. This account is a legitimate-looking failure pattern, not a verdict."},
		{ID: "shared_device", Account: "acct_kim", ExpectedRules: []string{"shared_device"}, OpensCase: true, Narrative: "A device already seen on another account. Not an identity conclusion."},
		{ID: "geographic_inconsistency", Account: "acct_quinn", ExpectedRules: []string{"geographic_inconsistency"}, OpensCase: true, Narrative: "Country change inside two hours."},
		{ID: "merchant_novelty", Account: "acct_wynn", ExpectedRules: []string{"merchant_novelty"}, OpensCase: true, Narrative: "First approved purchase at a merchant, above the amount floor."},
		{ID: "takeover_indicators", Account: "acct_ren", ExpectedRules: []string{"amount_deviation", "geographic_inconsistency", "merchant_novelty", "repeated_failures"}, OpensCase: true, Narrative: "Declines, a new device, a country change, and a large new-merchant purchase. Benign travel remains possible."},
		{ID: "legitimate_travel", Account: "acct_sam", ExpectedRules: []string{}, Legitimate: true, OpensCase: false, Narrative: "A later purchase in another city of the same country, below the amount rule."},
		{ID: "demo_mixed_devices", Account: "acct_demo", ExpectedRules: []string{"velocity"}, OpensCase: true, Narrative: "Rapid purchases plus a second device. A new phone or household member is a plausible benign alternative."},
	}
}

func Run(ctx context.Context, app *store.Store, admin *pgxpool.Pool, password string, reset bool) error {
	if reset {
		if admin == nil {
			return domain.ErrInvalid
		}
		if err := store.TruncateBusiness(ctx, admin); err != nil {
			return err
		}
	}
	if password == "" {
		password = "synthetic-demo-pass"
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	if err := app.EnsureTenant(ctx, TenantNorthwind, "Northwind Demo"); err != nil {
		return err
	}
	if err := app.EnsureTenant(ctx, TenantSouthbank, "Southbank Demo"); err != nil {
		return err
	}
	users := []store.User{
		{ID: "usr_nw_investigator", TenantID: TenantNorthwind, Email: "investigator@northwind.demo", Name: "Nia Investigator", Role: domain.RoleInvestigator, PasswordHash: hash},
		{ID: "usr_nw_senior", TenantID: TenantNorthwind, Email: "senior@northwind.demo", Name: "Sam Senior", Role: domain.RoleSenior, PasswordHash: hash},
		{ID: "usr_nw_viewer", TenantID: TenantNorthwind, Email: "viewer@northwind.demo", Name: "Val Viewer", Role: domain.RoleViewer, PasswordHash: hash},
		{ID: "usr_nw_admin", TenantID: TenantNorthwind, Email: "admin@northwind.demo", Name: "Ada Admin", Role: domain.RoleAdmin, PasswordHash: hash},
		{ID: "usr_sb_investigator", TenantID: TenantSouthbank, Email: "investigator@southbank.demo", Name: "Sol Investigator", Role: domain.RoleInvestigator, PasswordHash: hash},
	}
	for _, u := range users {
		if err := app.UpsertUser(ctx, u); err != nil {
			return err
		}
	}
	svc := &ingestion.Service{Store: app, Rules: rules.DefaultConfig()}
	nw := domain.Principal{UserID: "usr_nw_investigator", TenantID: TenantNorthwind, Email: "investigator@northwind.demo", Role: domain.RoleInvestigator}
	for _, event := range northwindEvents() {
		if _, err := svc.Ingest(ctx, nw, event); err != nil && err != domain.ErrConflict {
			return err
		}
	}
	sb := domain.Principal{UserID: "usr_sb_investigator", TenantID: TenantSouthbank, Email: "investigator@southbank.demo", Role: domain.RoleInvestigator}
	_, err = svc.Ingest(ctx, sb, ev("sb_1", "acct_south", "South Account", day(3), 1800, "approved", "mer_cafe", "Harbor Coffee", "food", "dev_south", "US", "Seattle", "WA", "tea"))
	if err == domain.ErrConflict {
		return nil
	}
	return err
}

func day(n int) time.Time {
	return time.Date(2026, 6, 15, 15, 0, 0, 0, time.UTC).AddDate(0, 0, -n)
}

func at(min int) time.Time {
	return time.Date(2026, 6, 15, 15, 0, 0, 0, time.UTC).Add(time.Duration(min) * time.Minute)
}

func ev(key, account, label string, when time.Time, amount int64, status, merchantID, merchantName, category, device, country, city, region, memo string) ingestion.Event {
	return ingestion.Event{
		IdempotencyKey: key, SourceSystem: "synthetic", SourceID: key, SchemaVersion: "1.0",
		OccurredAt: when, AccountExternalID: account, AccountLabel: label, AmountMinor: amount,
		Currency: "USD", Status: status, Memo: memo,
		Merchant: &ingestion.Merchant{ExternalID: merchantID, Name: merchantName, Category: category, Country: country},
		Device:   &ingestion.Device{ExternalID: device},
		Location: &ingestion.Location{Country: country, Region: region, City: city},
	}
}

func northwindEvents() []ingestion.Event {
	var out []ingestion.Event
	// Ordinary activity.
	out = append(out,
		ev("ava_1", "acct_ava", "Ava Example", day(12), 2200, "approved", "mer_cafe", "Harbor Coffee", "food", "dev_ava", "US", "Oakland", "CA", "coffee"),
		ev("ava_2", "acct_ava", "Ava Example", day(6), 2400, "approved", "mer_cafe", "Harbor Coffee", "food", "dev_ava", "US", "Oakland", "CA", "coffee"),
		ev("ava_3", "acct_ava", "Ava Example", day(1), 1800, "approved", "mer_cafe", "Harbor Coffee", "food", "dev_ava", "US", "Oakland", "CA", "coffee"),
	)
	// Velocity.
	for i := 0; i < 6; i++ {
		out = append(out, ev("milo_"+string(rune('a'+i)), "acct_milo", "Milo Example", at(i), 1500, "approved", "mer_cafe", "Harbor Coffee", "food", "dev_milo", "US", "Oakland", "CA", "repeat purchase"))
	}
	// Amount deviation against a seen merchant.
	for i := 0; i < 4; i++ {
		out = append(out, ev("lee_"+string(rune('a'+i)), "acct_lee", "Lee Example", day(10-i), 2000, "approved", "mer_lee", "Lee Market", "grocery", "dev_lee", "US", "Oakland", "CA", "groceries"))
	}
	out = append(out, ev("lee_big", "acct_lee", "Lee Example", at(30), 40000, "approved", "mer_lee", "Lee Market", "grocery", "dev_lee", "US", "Oakland", "CA", "large basket"))
	// Repeated declines.
	for i := 0; i < 3; i++ {
		out = append(out, ev("noor_"+string(rune('a'+i)), "acct_noor", "Noor Example", at(40+i), 5000, "declined", "mer_cafe", "Harbor Coffee", "food", "dev_noor", "US", "Oakland", "CA", "card declined"))
	}
	// Shared device. The second account observes the overlap.
	out = append(out,
		ev("sam_device", "acct_park", "Park Example", at(50), 2100, "approved", "mer_cafe", "Harbor Coffee", "food", "dev_shared", "US", "Oakland", "CA", "snack"),
		ev("kim_device", "acct_kim", "Kim Example", at(55), 2300, "approved", "mer_cafe", "Harbor Coffee", "food", "dev_shared", "US", "Oakland", "CA", "snack"),
	)
	// Geography.
	out = append(out,
		ev("quinn_us", "acct_quinn", "Quinn Example", at(60), 3000, "approved", "mer_cafe", "Harbor Coffee", "food", "dev_quinn", "US", "Oakland", "CA", "morning"),
		ev("quinn_de", "acct_quinn", "Quinn Example", at(80), 3200, "approved", "mer_cafe_de", "Elbe Coffee", "food", "dev_quinn", "DE", "Hamburg", "HH", "later"),
	)
	// New merchant above the floor. No prior baseline, so amount deviation stays quiet.
	out = append(out, ev("wynn_1", "acct_wynn", "Wynn Example", at(90), 80000, "approved", "mer_electronics", "Northline Electronics", "electronics", "dev_wynn", "US", "Oakland", "CA", "headset"))
	// Takeover indicators.
	for i := 0; i < 4; i++ {
		out = append(out, ev("ren_base_"+string(rune('a'+i)), "acct_ren", "Ren Example", day(8-i), 2000, "approved", "mer_cafe", "Harbor Coffee", "food", "dev_ren_home", "US", "Oakland", "CA", "usual"))
	}
	for i := 0; i < 3; i++ {
		out = append(out, ev("ren_dec_"+string(rune('a'+i)), "acct_ren", "Ren Example", at(100+i), 7000, "declined", "mer_cafe", "Harbor Coffee", "food", "dev_ren_new", "US", "Oakland", "CA", "declined"))
	}
	out = append(out, ev("ren_large", "acct_ren", "Ren Example", at(106), 90000, "approved", "mer_electronics", "Northline Electronics", "electronics", "dev_ren_new", "DE", "Hamburg", "HH", "new device purchase"))
	// Legitimate travel: same country, modest increase, outside short windows.
	for i := 0; i < 4; i++ {
		out = append(out, ev("travel_"+string(rune('a'+i)), "acct_sam", "Sam Example", day(9-i), 2500, "approved", "mer_cafe", "Harbor Coffee", "food", "dev_sam", "US", "Oakland", "CA", "usual"))
	}
	out = append(out, ev("travel_later", "acct_sam", "Sam Example", day(1), 3000, "approved", "mer_cafe", "Harbor Coffee", "food", "dev_sam", "US", "Seattle", "WA", "trip coffee"))
	// Demo path: velocity plus a second device inside the same window.
	for i := 0; i < 4; i++ {
		out = append(out, ev("demo_base_"+string(rune('a'+i)), "acct_demo", "Demo Example", day(7-i), 2500, "approved", "mer_cafe", "Harbor Coffee", "food", "dev_demo_home", "US", "Oakland", "CA", "baseline"))
	}
	for i := 0; i < 5; i++ {
		out = append(out, ev("demo_fast_"+string(rune('a'+i)), "acct_demo", "Demo Example", at(120+i), 1800, "approved", "mer_cafe", "Harbor Coffee", "food", "dev_demo_home", "US", "Oakland", "CA", "rapid"))
	}
	out = append(out, ev("demo_other_device", "acct_demo", "Demo Example", at(126), 4500, "approved", "mer_cafe", "Harbor Coffee", "food", "dev_demo_travel", "US", "Oakland", "CA", "second device"))
	return out
}
