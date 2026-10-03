package ingestion

import (
	"fmt"
	"regexp"
	"time"
	"unicode/utf8"
)

var idPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,64}$`)

type Merchant struct {
	ExternalID string `json:"external_id"`
	Name       string `json:"name"`
	Category   string `json:"category"`
	Country    string `json:"country"`
}

type Device struct {
	ExternalID string `json:"external_id"`
}

type Location struct {
	Country string `json:"country"`
	Region  string `json:"region"`
	City    string `json:"city"`
}

type Event struct {
	IdempotencyKey    string    `json:"idempotency_key"`
	SourceSystem      string    `json:"source_system"`
	SourceID          string    `json:"source_id"`
	SchemaVersion     string    `json:"schema_version"`
	OccurredAt        time.Time `json:"occurred_at"`
	AccountExternalID string    `json:"account_external_id"`
	AccountLabel      string    `json:"account_label"`
	Merchant          *Merchant `json:"merchant"`
	Device            *Device   `json:"device"`
	Location          *Location `json:"location"`
	AmountMinor       int64     `json:"amount_minor"`
	Currency          string    `json:"currency"`
	Status            string    `json:"status"`
	Memo              string    `json:"memo"`
}

func Validate(e Event, now time.Time) string {
	if !idPattern.MatchString(e.IdempotencyKey) {
		return "idempotency_key must be 1-64 characters of letters, numbers, and . _ : -"
	}
	if !idPattern.MatchString(e.SourceSystem) || !idPattern.MatchString(e.SourceID) {
		return "source_system and source_id are required opaque identifiers"
	}
	if e.SchemaVersion == "" || len(e.SchemaVersion) > 16 {
		return "schema_version is required"
	}
	if !idPattern.MatchString(e.AccountExternalID) {
		return "account_external_id is required"
	}
	if e.OccurredAt.IsZero() {
		return "occurred_at is required"
	}
	if e.OccurredAt.Before(now.AddDate(-10, 0, 0)) || e.OccurredAt.After(now.Add(24*time.Hour)) {
		return "occurred_at is outside the accepted window"
	}
	if e.AmountMinor < 0 || e.AmountMinor > 1_000_000_000_000 {
		return "amount_minor is out of range"
	}
	if len(e.Currency) != 3 {
		return "currency must be a 3-letter code"
	}
	switch e.Status {
	case "approved", "declined", "pending":
	default:
		return "status must be approved, declined, or pending"
	}
	if utf8.RuneCountInString(e.Memo) > 500 {
		return "memo exceeds 500 characters"
	}
	if utf8.RuneCountInString(e.AccountLabel) > 80 {
		return "account_label exceeds 80 characters"
	}
	if e.Merchant != nil {
		if e.Merchant.ExternalID != "" && !idPattern.MatchString(e.Merchant.ExternalID) {
			return "merchant.external_id is invalid"
		}
		if utf8.RuneCountInString(e.Merchant.Name) > 80 {
			return "merchant.name exceeds 80 characters"
		}
	}
	if e.Device != nil && e.Device.ExternalID != "" && !idPattern.MatchString(e.Device.ExternalID) {
		return "device.external_id is invalid"
	}
	if e.Location != nil && e.Location.Country != "" && len(e.Location.Country) != 2 {
		return "location.country must be 2 letters when set"
	}
	return ""
}

func locationKey(l *Location) string {
	if l == nil {
		return ""
	}
	return fmt.Sprintf("%s|%s|%s", l.Country, l.Region, l.City)
}
