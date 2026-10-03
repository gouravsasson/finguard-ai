package ingestion

import (
	"context"
	"encoding/json"
	"time"

	"finguard.ai/internal/domain"
	"finguard.ai/internal/idgen"
	"finguard.ai/internal/redact"
	"finguard.ai/internal/rules"
	"finguard.ai/internal/store"
)

type Service struct {
	Store     *store.Store
	Rules     rules.Config
	Retention time.Duration
}

type Result struct {
	IdempotencyKey   string   `json:"idempotency_key"`
	Status           string   `json:"status"`
	TransactionID    string   `json:"transaction_id,omitempty"`
	CaseID           string   `json:"case_id,omitempty"`
	SignalCodes      []string `json:"signal_codes,omitempty"`
	QuarantineReason string   `json:"quarantine_reason,omitempty"`
	RawEventID       string   `json:"raw_event_id,omitempty"`
}

func (s *Service) QuarantineBytes(ctx context.Context, p domain.Principal, body []byte, reason string) error {
	if len(body) > 4000 {
		body = body[:4000]
	}
	body = []byte(redact.Text(string(body)))
	hash := idgen.Hash(string(body))
	key := "bad_" + hash[:40]
	sc, err := s.Store.Begin(ctx, p.TenantID)
	if err != nil {
		return err
	}
	defer sc.Rollback()
	retention := s.Retention
	if retention == 0 {
		retention = 90 * 24 * time.Hour
	}
	if _, err := sc.PutRaw(ctx, idgen.New("raw_"), key, "unknown", "", "0", hash, domain.RawQuarantined, reason, body, time.Now().UTC().Add(retention)); err != nil {
		return err
	}
	return sc.Commit(ctx)
}

func (s *Service) Ingest(ctx context.Context, p domain.Principal, event Event) (Result, error) {
	now := time.Now().UTC()
	reason := Validate(event, now)
	event.Memo = redact.Text(event.Memo)
	if event.Merchant != nil {
		event.Merchant.Name = redact.Text(event.Merchant.Name)
	}
	raw, err := json.Marshal(event)
	if err != nil {
		return Result{}, err
	}
	hash := idgen.Hash(string(raw))
	retention := s.Retention
	if retention == 0 {
		retention = 90 * 24 * time.Hour
	}
	sc, err := s.Store.Begin(ctx, p.TenantID)
	if err != nil {
		return Result{}, err
	}
	defer sc.Rollback()

	status := domain.RawAccepted
	if reason != "" {
		status = domain.RawQuarantined
	}
	rawID := idgen.New("raw_")
	put, err := sc.PutRaw(ctx, rawID, event.IdempotencyKey, blank(event.SourceSystem, "unknown"), event.SourceID, blank(event.SchemaVersion, "0"), hash, status, reason, raw, now.Add(retention))
	if err != nil {
		return Result{}, err
	}
	if !put.Inserted {
		if put.PayloadHash != hash {
			return Result{IdempotencyKey: event.IdempotencyKey, Status: "conflict", RawEventID: put.ID}, domain.ErrConflict
		}
		res := Result{IdempotencyKey: event.IdempotencyKey, Status: "replayed", RawEventID: put.ID}
		if put.Status == domain.RawQuarantined {
			res.Status = "quarantined"
			res.QuarantineReason = reason
			if err := sc.Commit(ctx); err != nil {
				return Result{}, err
			}
			return res, nil
		}
		txn, err := sc.TransactionBySource(ctx, event.SourceSystem, event.SourceID)
		if err != nil && err != domain.ErrNotFound {
			return Result{}, err
		}
		res.TransactionID = txn.ID
		if cse, err := sc.ActiveCase(ctx, txn.AccountID); err == nil {
			res.CaseID = cse.ID
		}
		if err := sc.Commit(ctx); err != nil {
			return Result{}, err
		}
		return res, nil
	}
	if reason != "" {
		if err := s.audit(ctx, sc, p, "event_quarantined", put.ID, reason); err != nil {
			return Result{}, err
		}
		if err := sc.Commit(ctx); err != nil {
			return Result{}, err
		}
		return Result{IdempotencyKey: event.IdempotencyKey, Status: "quarantined", QuarantineReason: reason, RawEventID: put.ID}, nil
	}

	label := event.AccountLabel
	if label == "" {
		label = event.AccountExternalID
	}
	accountID, err := sc.UpsertAccount(ctx, idgen.New("acct_"), event.AccountExternalID, label, event.Currency)
	if err != nil {
		return Result{}, err
	}
	var merchantID, deviceID, deviceExternal, locationID string
	if event.Merchant != nil {
		merchantID, err = sc.UpsertMerchant(ctx, idgen.New("mer_"), event.Merchant.ExternalID, event.Merchant.Name, event.Merchant.Category, event.Merchant.Country)
		if err != nil {
			return Result{}, err
		}
	}
	if event.Device != nil {
		deviceExternal = event.Device.ExternalID
		deviceID, err = sc.UpsertDevice(ctx, idgen.New("dev_"), event.Device.ExternalID)
		if err != nil {
			return Result{}, err
		}
	}
	country, region, city := "", "", ""
	if event.Location != nil {
		country, region, city = event.Location.Country, event.Location.Region, event.Location.City
		locationID, err = sc.UpsertLocation(ctx, idgen.New("loc_"), country, region, city, locationKey(event.Location))
		if err != nil {
			return Result{}, err
		}
	}
	txn := domain.Transaction{
		ID: idgen.New("txn_"), TenantID: p.TenantID, RawEventID: put.ID, SourceSystem: event.SourceSystem,
		SourceID: event.SourceID, AccountID: accountID, MerchantID: merchantID, DeviceID: deviceID,
		LocationID: locationID, LocationCountry: country, AmountMinor: event.AmountMinor, Currency: event.Currency,
		Status: event.Status, OccurredAt: event.OccurredAt.UTC(), Memo: event.Memo, SchemaVersion: event.SchemaVersion,
	}
	if err := sc.InsertTransaction(ctx, txn); err != nil {
		return Result{}, err
	}
	histRows, err := sc.AccountHistory(ctx, accountID, txn.OccurredAt)
	if err != nil {
		return Result{}, err
	}
	seen, err := sc.MerchantSeen(ctx, accountID, merchantID, txn.ID)
	if err != nil {
		return Result{}, err
	}
	deviceAccounts, err := sc.DeviceAccounts(ctx, deviceID)
	if err != nil {
		return Result{}, err
	}
	current := rules.Transaction{
		ID: txn.ID, AccountID: accountID, MerchantID: merchantID, DeviceID: deviceID, Country: country,
		AmountMinor: txn.AmountMinor, Status: txn.Status, OccurredAt: txn.OccurredAt,
	}
	var accountHist []rules.Transaction
	byID := map[string]store.HistTxn{}
	for _, row := range histRows {
		byID[row.ID] = row
		accountHist = append(accountHist, rules.Transaction{
			ID: row.ID, AccountID: row.AccountID, MerchantID: row.MerchantID, DeviceID: row.DeviceID,
			Country: row.Country, AmountMinor: row.AmountMinor, Status: row.Status, OccurredAt: row.OccurredAt,
		})
	}
	signals := rules.Evaluate(s.Rules, current, rules.History{
		Account: accountHist, DeviceAccountIDs: deviceAccounts, MerchantSeenBefore: seen,
	})
	res := Result{IdempotencyKey: event.IdempotencyKey, Status: "accepted", TransactionID: txn.ID, RawEventID: put.ID}
	if len(signals) > 0 {
		caseID, err := s.ensureCase(ctx, sc, accountID, event.AccountExternalID)
		if err != nil {
			return Result{}, err
		}
		res.CaseID = caseID
		var evidenceIDs []string
		for _, sig := range signals {
			ids, err := s.writeSignal(ctx, sc, p, caseID, txn, event, sig, byID, deviceExternal, len(deviceAccounts))
			if err != nil {
				return Result{}, err
			}
			evidenceIDs = append(evidenceIDs, ids...)
			res.SignalCodes = append(res.SignalCodes, sig.RuleCode)
			row := domain.Signal{
				ID: idgen.New("sig_"), CaseID: caseID, TransactionID: txn.ID, RuleCode: sig.RuleCode,
				RuleVersion: sig.RuleVersion, Explanation: sig.Explanation, Threshold: sig.Threshold,
				ObservedValue: sig.Observed, Inputs: sig.Inputs, EvidenceIDs: ids,
			}
			if _, err := sc.InsertSignal(ctx, row); err != nil {
				return Result{}, err
			}
		}
		if err := sc.LinkTransaction(ctx, caseID, txn.ID); err != nil {
			return Result{}, err
		}
		if err := sc.TouchCaseSignals(ctx, caseID); err != nil {
			return Result{}, err
		}
		if _, err := sc.InsertJob(ctx, domain.Job{
			ID: idgen.New("job_"), Type: "ai_report",
			IdempotencyKey: "ai_report:" + caseID + ":" + idgen.Hash(join(evidenceIDs)),
			Payload:        map[string]any{"case_id": caseID},
			MaxAttempts:    5,
		}); err != nil {
			return Result{}, err
		}
	}
	if err := s.audit(ctx, sc, p, "event_ingested", txn.ID, ""); err != nil {
		return Result{}, err
	}
	if err := sc.Commit(ctx); err != nil {
		return Result{}, err
	}
	return res, nil
}

func (s *Service) ensureCase(ctx context.Context, sc *store.Scope, accountID, external string) (string, error) {
	existing, err := sc.ActiveCase(ctx, accountID)
	if err == nil {
		return existing.ID, nil
	}
	if err != domain.ErrNotFound {
		return "", err
	}
	c := domain.Case{
		ID: idgen.New("case_"), AccountID: accountID, Priority: domain.TriageMedium,
		Title: "Review activity on " + external,
	}
	if err := sc.InsertCase(ctx, c); err != nil {
		if err == domain.ErrConflict {
			existing, err = sc.ActiveCase(ctx, accountID)
			return existing.ID, err
		}
		return "", err
	}
	if err := sc.InsertAudit(ctx, domain.AuditEvent{
		ID: idgen.New("aud_"), ActorLabel: "system", Action: "case_opened",
		EntityType: "case", EntityID: c.ID, Metadata: map[string]any{"account": external},
	}); err != nil {
		return "", err
	}
	return c.ID, nil
}

func (s *Service) writeSignal(ctx context.Context, sc *store.Scope, p domain.Principal, caseID string, txn domain.Transaction, event Event, sig rules.Signal, byID map[string]store.HistTxn, deviceExternal string, deviceAccounts int) ([]string, error) {
	var ids []string
	related := sig.RelatedTxnIDs
	if len(related) > 15 {
		related = related[:15]
	}
	for _, rel := range related {
		row, ok := byID[rel]
		if !ok {
			continue
		}
		key := "txn:" + row.SourceSystem + ":" + row.SourceID
		id := evidenceID(p.TenantID, key)
		summary := row.SourceSystem + ":" + row.SourceID + " " + row.Status + " " + itoa(row.AmountMinor) + " " + row.Country
		if err := insertEvidence(ctx, sc, domain.Evidence{
			ID: id, CaseID: caseID, StableKey: key, Version: 1, Kind: "transaction_record",
			Source: row.SourceSystem, Classification: "confidential",
			Payload: map[string]any{
				"summary": summary, "source_id": row.SourceID, "amount_minor": row.AmountMinor,
				"status": row.Status, "country": row.Country, "occurred_at": row.OccurredAt,
			},
			Provenance: map[string]any{"source_system": row.SourceSystem, "source_id": row.SourceID, "method": "ingestion"},
		}); err != nil {
			return nil, err
		}
		if err := sc.LinkTransaction(ctx, caseID, rel); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	ruleKey := "rule:" + txn.SourceSystem + ":" + txn.SourceID + ":" + sig.RuleCode + ":" + sig.RuleVersion
	ruleID := evidenceID(p.TenantID, ruleKey)
	if err := insertEvidence(ctx, sc, domain.Evidence{
		ID: ruleID, CaseID: caseID, StableKey: ruleKey, Version: 1, Kind: "rule_evaluation",
		Source: "deterministic-rules", Classification: "confidential",
		Payload: map[string]any{
			"summary": sig.Explanation, "rule_code": sig.RuleCode, "rule_version": sig.RuleVersion,
			"threshold": sig.Threshold, "observed": sig.Observed,
		},
		Provenance: map[string]any{"method": "deterministic-rule", "rule_code": sig.RuleCode, "rule_version": sig.RuleVersion},
	}); err != nil {
		return nil, err
	}
	ids = append(ids, ruleID)
	if sig.RuleCode == "shared_device" {
		key := "device:" + deviceExternal + ":" + event.AccountExternalID
		id := evidenceID(p.TenantID, key)
		if err := insertEvidence(ctx, sc, domain.Evidence{
			ID: id, CaseID: caseID, StableKey: key, Version: 1, Kind: "device_observation",
			Source: "deterministic-rules", Classification: "confidential",
			Payload: map[string]any{
				"summary":            "Device " + deviceExternal + " was observed on multiple accounts. This is not an identity conclusion.",
				"device_external_id": deviceExternal,
				"distinct_accounts":  deviceAccounts,
				"identity_inference": false,
			},
			Provenance: map[string]any{"method": "deterministic-rule", "rule_code": "shared_device"},
		}); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return unique(ids), nil
}

func insertEvidence(ctx context.Context, sc *store.Scope, e domain.Evidence) error {
	body, err := json.Marshal(e.Payload)
	if err != nil {
		return err
	}
	e.ContentHash = idgen.Hash(string(body))
	e.RetentionUntil = time.Now().UTC().Add(365 * 24 * time.Hour)
	if e.Version == 0 {
		e.Version = 1
	}
	return sc.InsertEvidence(ctx, e)
}

func (s *Service) audit(ctx context.Context, sc *store.Scope, p domain.Principal, action, entityID, rationale string) error {
	return sc.InsertAudit(ctx, domain.AuditEvent{
		ID: idgen.New("aud_"), ActorID: p.UserID, ActorLabel: p.Email, Action: action,
		EntityType: "transaction", EntityID: entityID, Rationale: rationale,
	})
}

func evidenceID(tenant, key string) string {
	return idgen.Stable("ev_", tenant+"|"+key)
}

func blank(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func itoa(n int64) string {
	return json.Number(intString(n)).String()
}

func intString(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func unique(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func join(ids []string) string {
	out := ""
	for _, id := range ids {
		out += id + "|"
	}
	return out
}
