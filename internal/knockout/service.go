package knockout

import (
	"context"
	"time"

	"finguard.ai/internal/rules"
	"finguard.ai/internal/store"
)

// Service loads one tenant's case and replays its signals. It does not update rows.
type Service struct {
	Store *store.Store
	Rules rules.Config
}

func (s *Service) ForCase(ctx context.Context, tenantID, caseID string) (Board, error) {
	cfg := s.Rules
	if cfg.VelocityMinCount == 0 {
		cfg = rules.DefaultConfig()
	}
	sc, err := s.Store.Begin(ctx, tenantID)
	if err != nil {
		return Board{}, err
	}
	defer sc.Rollback()
	cse, err := sc.GetCase(ctx, caseID)
	if err != nil {
		return Board{}, err
	}
	signals, err := sc.ListSignals(ctx, caseID)
	if err != nil {
		return Board{}, err
	}
	evidence, err := sc.ListEvidence(ctx, caseID)
	if err != nil {
		return Board{}, err
	}
	history, err := sc.AccountHistory(ctx, cse.AccountID, time.Now().UTC().AddDate(1, 0, 0))
	if err != nil {
		return Board{}, err
	}
	bySource := map[string]string{}
	for _, item := range evidence {
		if item.Kind != "transaction_record" {
			continue
		}
		if sourceID, ok := item.Payload["source_id"].(string); ok && sourceID != "" {
			bySource[sourceID] = item.ID
		}
	}
	byTxn := map[string]string{}
	var account []rules.Transaction
	rows := map[string]store.HistTxn{}
	for _, row := range history {
		rows[row.ID] = row
		account = append(account, rules.Transaction{
			ID: row.ID, AccountID: row.AccountID, MerchantID: row.MerchantID, DeviceID: row.DeviceID,
			Country: row.Country, AmountMinor: row.AmountMinor, Status: row.Status, OccurredAt: row.OccurredAt,
		})
		if ev, ok := bySource[row.SourceID]; ok {
			byTxn[row.ID] = ev
		}
	}
	board := Board{
		Method:      "deterministic-replay",
		RuleVersion: rules.Version,
		Limitation:  "These are hypothetical replays on a copy of the account history. No transaction was changed, deleted, or reversed, and a cleared replay is not a finding that the activity was legitimate.",
		Signals:     []SignalReplay{},
	}
	for _, sig := range signals {
		row, ok := rows[sig.TransactionID]
		if !ok {
			board.Signals = append(board.Signals, SignalReplay{
				SignalID: sig.ID, RuleCode: sig.RuleCode, TransactionID: sig.TransactionID,
				Explanation: sig.Explanation, Note: "The triggering transaction is not in the account history available for replay.",
				Attempts: []Attempt{},
			})
			continue
		}
		trigger := rules.Transaction{
			ID: row.ID, AccountID: row.AccountID, MerchantID: row.MerchantID, DeviceID: row.DeviceID,
			Country: row.Country, AmountMinor: row.AmountMinor, Status: row.Status, OccurredAt: row.OccurredAt,
		}
		deviceAccounts, err := sc.DeviceAccounts(ctx, row.DeviceID)
		if err != nil {
			return Board{}, err
		}
		seen, err := sc.MerchantSeen(ctx, row.AccountID, row.MerchantID, row.ID)
		if err != nil {
			return Board{}, err
		}
		replay := Replay(cfg, sig.RuleCode, trigger, rules.History{
			Account: account, DeviceAccountIDs: deviceAccounts, MerchantSeenBefore: seen,
		})
		replay.SignalID = sig.ID
		if replay.Explanation == "" {
			replay.Explanation = sig.Explanation
		}
		ruleEvidence := append([]string(nil), sig.EvidenceIDs...)
		for i := range replay.Attempts {
			replay.Attempts[i].EvidenceIDs = cite(replay.Attempts[i].TransactionIDs, byTxn, ruleEvidence)
		}
		board.Signals = append(board.Signals, replay)
	}
	board.Reading = ComposeReading(board.Signals)
	return board, nil
}

func cite(txnIDs []string, byTxn map[string]string, ruleEvidence []string) []string {
	var ids []string
	for _, id := range txnIDs {
		if ev := byTxn[id]; ev != "" {
			ids = append(ids, ev)
		}
	}
	ids = append(ids, ruleEvidence...)
	return unique(ids)
}
