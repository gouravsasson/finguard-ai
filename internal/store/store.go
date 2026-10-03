// Package store is the PostgreSQL boundary. Tenant-scoped work runs on a
// connection with app.tenant_id set so row-level security applies.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"finguard.ai/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	Pool *pgxpool.Pool
}

func Open(ctx context.Context, url string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, `SET TIME ZONE 'UTC'`)
		return err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{Pool: pool}, nil
}

func (s *Store) Close() {
	s.Pool.Close()
}

type Scope struct {
	TenantID string
	Tx       pgx.Tx
	conn     *pgxpool.Conn
	done     bool
}

func (s *Store) Begin(ctx context.Context, tenantID string) (*Scope, error) {
	if tenantID == "" {
		return nil, errors.New("missing tenant")
	}
	conn, err := s.Pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	if _, err = conn.Exec(ctx, `SELECT set_config('app.tenant_id', $1, false)`, tenantID); err != nil {
		conn.Release()
		return nil, err
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		_, _ = conn.Exec(context.Background(), `SELECT set_config('app.tenant_id', '', false)`)
		conn.Release()
		return nil, err
	}
	return &Scope{TenantID: tenantID, Tx: tx, conn: conn}, nil
}

func (sc *Scope) Commit(ctx context.Context) error {
	if sc == nil || sc.done {
		return errors.New("scope closed")
	}
	sc.done = true
	err := sc.Tx.Commit(ctx)
	if err != nil {
		_ = sc.Tx.Rollback(context.Background())
	}
	sc.release()
	return err
}

func (sc *Scope) Rollback() {
	if sc == nil || sc.done {
		return
	}
	sc.done = true
	_ = sc.Tx.Rollback(context.Background())
	sc.release()
}

func (sc *Scope) release() {
	_, _ = sc.conn.Exec(context.Background(), `SELECT set_config('app.tenant_id', '', false)`)
	sc.conn.Release()
}

func isUnique(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func toJSON(v any) ([]byte, error) {
	if v == nil {
		return []byte("null"), nil
	}
	return json.Marshal(v)
}

func scanMap(b []byte) (map[string]any, error) {
	if len(b) == 0 || string(b) == "null" {
		return map[string]any{}, nil
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}

func scanStrings(b []byte) ([]string, error) {
	if len(b) == 0 || string(b) == "null" {
		return []string{}, nil
	}
	var s []string
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	if s == nil {
		s = []string{}
	}
	return s, nil
}

type User struct {
	ID           string
	TenantID     string
	Email        string
	Name         string
	Role         string
	PasswordHash string
}

func (s *Store) EnsureTenant(ctx context.Context, id, name string) error {
	_, err := s.Pool.Exec(ctx, `INSERT INTO tenants (id, name) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name`, id, name)
	return err
}

func (s *Store) UpsertUser(ctx context.Context, u User) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO users (id, tenant_id, email, display_name, role, password_hash)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (email) DO UPDATE SET
		  tenant_id = EXCLUDED.tenant_id,
		  display_name = EXCLUDED.display_name,
		  role = EXCLUDED.role,
		  password_hash = EXCLUDED.password_hash`,
		u.ID, u.TenantID, u.Email, u.Name, u.Role, u.PasswordHash)
	return err
}

func (s *Store) UserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := s.Pool.QueryRow(ctx, `SELECT id, tenant_id, email, display_name, role, password_hash FROM users WHERE email = $1`, email).
		Scan(&u.ID, &u.TenantID, &u.Email, &u.Name, &u.Role, &u.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, domain.ErrNotFound
	}
	return u, err
}

func (s *Store) CreateSession(ctx context.Context, id, userID, tokenHash string, exp time.Time) error {
	_, err := s.Pool.Exec(ctx, `INSERT INTO sessions (id, user_id, token_hash, expires_at) VALUES ($1, $2, $3, $4)`, id, userID, tokenHash, exp)
	return err
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}

func (s *Store) PrincipalByToken(ctx context.Context, tokenHash string) (domain.Principal, error) {
	var p domain.Principal
	err := s.Pool.QueryRow(ctx, `
		SELECT u.id, u.tenant_id, u.email, u.display_name, u.role
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > now()`, tokenHash).
		Scan(&p.UserID, &p.TenantID, &p.Email, &p.Name, &p.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Principal{}, domain.ErrUnauthenticated
	}
	return p, err
}

type RawResult struct {
	ID          string
	PayloadHash string
	Status      string
	Inserted    bool
}

func (sc *Scope) PutRaw(ctx context.Context, id, idem, sourceSystem, sourceID, schemaVersion, payloadHash, status, reason string, payload []byte, retention time.Time) (RawResult, error) {
	var got string
	err := sc.Tx.QueryRow(ctx, `
		INSERT INTO raw_events (id, tenant_id, idempotency_key, source_system, source_id, schema_version, payload, payload_hash, status, quarantine_reason, retention_until)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (tenant_id, idempotency_key) DO NOTHING
		RETURNING id`,
		id, sc.TenantID, idem, sourceSystem, sourceID, schemaVersion, payload, payloadHash, status, reason, retention).Scan(&got)
	if err == nil {
		return RawResult{ID: got, PayloadHash: payloadHash, Status: status, Inserted: true}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return RawResult{}, err
	}
	var existing RawResult
	err = sc.Tx.QueryRow(ctx, `SELECT id, payload_hash, status FROM raw_events WHERE tenant_id = $1 AND idempotency_key = $2`, sc.TenantID, idem).
		Scan(&existing.ID, &existing.PayloadHash, &existing.Status)
	return existing, err
}

func (sc *Scope) UpsertAccount(ctx context.Context, id, external, label, currency string) (string, error) {
	var got string
	err := sc.Tx.QueryRow(ctx, `
		INSERT INTO accounts (id, tenant_id, external_id, display_label, currency)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tenant_id, external_id) DO UPDATE SET display_label = EXCLUDED.display_label
		RETURNING id`, id, sc.TenantID, external, label, currency).Scan(&got)
	return got, err
}

func (sc *Scope) UpsertMerchant(ctx context.Context, id, external, name, category, country string) (string, error) {
	if external == "" {
		return "", nil
	}
	var got string
	err := sc.Tx.QueryRow(ctx, `
		INSERT INTO merchants (id, tenant_id, external_id, name, category, country)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (tenant_id, external_id) DO UPDATE SET name = EXCLUDED.name
		RETURNING id`, id, sc.TenantID, external, name, category, country).Scan(&got)
	return got, err
}

func (sc *Scope) UpsertDevice(ctx context.Context, id, external string) (string, error) {
	if external == "" {
		return "", nil
	}
	var got string
	err := sc.Tx.QueryRow(ctx, `
		INSERT INTO devices (id, tenant_id, external_id) VALUES ($1, $2, $3)
		ON CONFLICT (tenant_id, external_id) DO UPDATE SET external_id = EXCLUDED.external_id
		RETURNING id`, id, sc.TenantID, external).Scan(&got)
	return got, err
}

func (sc *Scope) UpsertLocation(ctx context.Context, id, country, region, city, key string) (string, error) {
	if key == "" || key == "||" {
		return "", nil
	}
	var got string
	err := sc.Tx.QueryRow(ctx, `
		INSERT INTO locations (id, tenant_id, country, region, city, natural_key)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (tenant_id, natural_key) DO UPDATE SET city = EXCLUDED.city
		RETURNING id`, id, sc.TenantID, country, region, city, key).Scan(&got)
	return got, err
}

func (sc *Scope) InsertTransaction(ctx context.Context, t domain.Transaction) error {
	tag, err := sc.Tx.Exec(ctx, `
		INSERT INTO transactions (
		  id, tenant_id, raw_event_id, source_system, source_id, account_id, merchant_id, device_id,
		  location_id, location_country, amount_minor, currency, status, occurred_at, memo, schema_version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		ON CONFLICT (tenant_id, source_system, source_id) DO NOTHING`,
		t.ID, sc.TenantID, t.RawEventID, t.SourceSystem, t.SourceID, t.AccountID, t.MerchantID, t.DeviceID,
		t.LocationID, t.LocationCountry, t.AmountMinor, t.Currency, t.Status, t.OccurredAt, t.Memo, t.SchemaVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrConflict
	}
	return nil
}

type HistTxn struct {
	ID           string
	AccountID    string
	MerchantID   string
	DeviceID     string
	Country      string
	AmountMinor  int64
	Status       string
	OccurredAt   time.Time
	SourceSystem string
	SourceID     string
}

func (sc *Scope) AccountHistory(ctx context.Context, accountID string, until time.Time) ([]HistTxn, error) {
	rows, err := sc.Tx.Query(ctx, `
		SELECT id, account_id, merchant_id, device_id, location_country, amount_minor, status, occurred_at, source_system, source_id
		FROM transactions
		WHERE tenant_id = $1 AND account_id = $2 AND occurred_at <= $3
		ORDER BY occurred_at ASC, id ASC`, sc.TenantID, accountID, until)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []HistTxn
	for rows.Next() {
		var h HistTxn
		if err := rows.Scan(&h.ID, &h.AccountID, &h.MerchantID, &h.DeviceID, &h.Country, &h.AmountMinor, &h.Status, &h.OccurredAt, &h.SourceSystem, &h.SourceID); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (sc *Scope) DeviceAccounts(ctx context.Context, deviceID string) ([]string, error) {
	if deviceID == "" {
		return nil, nil
	}
	rows, err := sc.Tx.Query(ctx, `SELECT DISTINCT account_id FROM transactions WHERE tenant_id = $1 AND device_id = $2`, sc.TenantID, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (sc *Scope) MerchantSeen(ctx context.Context, accountID, merchantID, exceptTxn string) (bool, error) {
	if merchantID == "" {
		return false, nil
	}
	var seen bool
	err := sc.Tx.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM transactions
		  WHERE tenant_id = $1 AND account_id = $2 AND merchant_id = $3 AND id <> $4
		)`, sc.TenantID, accountID, merchantID, exceptTxn).Scan(&seen)
	return seen, err
}

func (sc *Scope) TransactionBySource(ctx context.Context, system, sourceID string) (domain.Transaction, error) {
	row := sc.Tx.QueryRow(ctx, txnSelect+` WHERE t.tenant_id = $1 AND t.source_system = $2 AND t.source_id = $3`, sc.TenantID, system, sourceID)
	return scanTxn(row)
}

const txnSelect = `
SELECT t.id, t.tenant_id, t.raw_event_id, t.source_system, t.source_id, t.account_id,
       COALESCE(a.external_id, ''), COALESCE(t.merchant_id, ''), COALESCE(m.name, ''),
       COALESCE(t.device_id, ''), COALESCE(d.external_id, ''), COALESCE(t.location_id, ''),
       COALESCE(t.location_country, ''),
       trim(both ' ' FROM concat_ws(', ', NULLIF(l.city, ''), NULLIF(l.region, ''), NULLIF(l.country, ''))),
       t.amount_minor, t.currency, t.status, t.occurred_at, t.memo, t.schema_version, t.created_at
FROM transactions t
LEFT JOIN accounts a ON a.id = t.account_id AND a.tenant_id = t.tenant_id
LEFT JOIN merchants m ON m.id = t.merchant_id AND m.tenant_id = t.tenant_id
LEFT JOIN devices d ON d.id = t.device_id AND d.tenant_id = t.tenant_id
LEFT JOIN locations l ON l.id = t.location_id AND l.tenant_id = t.tenant_id`

func scanTxn(row pgx.Row) (domain.Transaction, error) {
	var t domain.Transaction
	err := row.Scan(
		&t.ID, &t.TenantID, &t.RawEventID, &t.SourceSystem, &t.SourceID, &t.AccountID,
		&t.AccountExternal, &t.MerchantID, &t.MerchantName, &t.DeviceID, &t.DeviceExternal,
		&t.LocationID, &t.LocationCountry, &t.LocationLabel, &t.AmountMinor, &t.Currency,
		&t.Status, &t.OccurredAt, &t.Memo, &t.SchemaVersion, &t.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Transaction{}, domain.ErrNotFound
	}
	return t, err
}

func (sc *Scope) ListTransactions(ctx context.Context, limit int) ([]domain.Transaction, error) {
	rows, err := sc.Tx.Query(ctx, txnSelect+` WHERE t.tenant_id = $1 ORDER BY t.occurred_at DESC, t.id DESC LIMIT $2`, sc.TenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Transaction
	for rows.Next() {
		t, err := scanTxn(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (sc *Scope) ActiveCase(ctx context.Context, accountID string) (domain.Case, error) {
	row := sc.Tx.QueryRow(ctx, caseSelect+` WHERE c.tenant_id = $1 AND c.account_id = $2 AND c.status IN ('open','in_review','escalated')`, sc.TenantID, accountID)
	return scanCase(row)
}

func (sc *Scope) InsertCase(ctx context.Context, c domain.Case) error {
	_, err := sc.Tx.Exec(ctx, `
		INSERT INTO cases (id, tenant_id, account_id, status, priority, title, version, signal_count, opened_at, updated_at)
		VALUES ($1, $2, $3, 'open', $4, $5, 1, 0, now(), now())`,
		c.ID, sc.TenantID, c.AccountID, c.Priority, c.Title)
	if isUnique(err) {
		return domain.ErrConflict
	}
	return err
}

func (sc *Scope) LinkTransaction(ctx context.Context, caseID, txnID string) error {
	_, err := sc.Tx.Exec(ctx, `
		INSERT INTO case_transactions (tenant_id, case_id, transaction_id) VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING`, sc.TenantID, caseID, txnID)
	return err
}

func (sc *Scope) TouchCaseSignals(ctx context.Context, caseID string) error {
	_, err := sc.Tx.Exec(ctx, `
		UPDATE cases SET
		  signal_count = sub.n,
		  priority = CASE WHEN sub.n >= 3 THEN 'high' WHEN sub.n >= 1 THEN 'medium' ELSE 'low' END,
		  version = version + 1,
		  updated_at = now()
		FROM (SELECT count(*)::int AS n FROM risk_signals WHERE tenant_id = $2 AND case_id = $1) sub
		WHERE cases.id = $1 AND cases.tenant_id = $2`, caseID, sc.TenantID)
	return err
}

const caseSelect = `
SELECT c.id, c.tenant_id, c.account_id, COALESCE(a.external_id, ''), c.status, c.priority, c.title,
       COALESCE(c.assignee_id, ''), c.version, c.signal_count, c.opened_at, c.updated_at, c.resolved_at,
       c.resolution, c.rationale
FROM cases c
LEFT JOIN accounts a ON a.id = c.account_id AND a.tenant_id = c.tenant_id`

func scanCase(row pgx.Row) (domain.Case, error) {
	var c domain.Case
	var resolved *time.Time
	err := row.Scan(&c.ID, &c.TenantID, &c.AccountID, &c.AccountExt, &c.Status, &c.Priority, &c.Title,
		&c.AssigneeID, &c.Version, &c.SignalCount, &c.OpenedAt, &c.UpdatedAt, &resolved, &c.Resolution, &c.Rationale)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Case{}, domain.ErrNotFound
	}
	c.ResolvedAt = resolved
	return c, err
}

func (sc *Scope) GetCase(ctx context.Context, id string) (domain.Case, error) {
	return scanCase(sc.Tx.QueryRow(ctx, caseSelect+` WHERE c.tenant_id = $1 AND c.id = $2`, sc.TenantID, id))
}

func (sc *Scope) ListCases(ctx context.Context, status string, cursor time.Time, cursorID string, limit int) ([]domain.Case, error) {
	rows, err := sc.Tx.Query(ctx, caseSelect+`
		WHERE c.tenant_id = $1
		  AND ($2 = '' OR c.status = $2)
		  AND ($3::timestamptz IS NULL OR (c.opened_at, c.id) < ($3, $4))
		ORDER BY c.opened_at DESC, c.id DESC
		LIMIT $5`, sc.TenantID, status, nullTime(cursor), cursorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Case
	for rows.Next() {
		c, err := scanCase(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

func (sc *Scope) UpdateCase(ctx context.Context, c domain.Case, prevVersion int) error {
	tag, err := sc.Tx.Exec(ctx, `
		UPDATE cases SET status = $1, assignee_id = NULLIF($2, ''), resolution = $3, rationale = $4,
		  resolved_at = $5, version = version + 1, updated_at = now()
		WHERE id = $6 AND tenant_id = $7 AND version = $8`,
		c.Status, c.AssigneeID, c.Resolution, c.Rationale, c.ResolvedAt, c.ID, sc.TenantID, prevVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		_, err := sc.GetCase(ctx, c.ID)
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		if err != nil {
			return err
		}
		return domain.ErrVersion
	}
	return nil
}

func (sc *Scope) InsertSignal(ctx context.Context, sig domain.Signal) (bool, error) {
	inputs, err := toJSON(sig.Inputs)
	if err != nil {
		return false, err
	}
	ev, err := toJSON(sig.EvidenceIDs)
	if err != nil {
		return false, err
	}
	tag, err := sc.Tx.Exec(ctx, `
		INSERT INTO risk_signals (
		  id, tenant_id, case_id, transaction_id, rule_code, rule_version, explanation,
		  threshold, observed_value, inputs, evidence_ids)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (tenant_id, transaction_id, rule_code, rule_version) DO NOTHING`,
		sig.ID, sc.TenantID, sig.CaseID, sig.TransactionID, sig.RuleCode, sig.RuleVersion,
		sig.Explanation, sig.Threshold, sig.ObservedValue, inputs, ev)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (sc *Scope) ListSignals(ctx context.Context, caseID string) ([]domain.Signal, error) {
	rows, err := sc.Tx.Query(ctx, `
		SELECT id, tenant_id, case_id, transaction_id, rule_code, rule_version, explanation,
		       threshold, observed_value, inputs, evidence_ids, created_at
		FROM risk_signals WHERE tenant_id = $1 AND case_id = $2 ORDER BY created_at ASC, rule_code ASC`, sc.TenantID, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Signal
	for rows.Next() {
		var sig domain.Signal
		var inputs, ev []byte
		if err := rows.Scan(&sig.ID, &sig.TenantID, &sig.CaseID, &sig.TransactionID, &sig.RuleCode, &sig.RuleVersion,
			&sig.Explanation, &sig.Threshold, &sig.ObservedValue, &inputs, &ev, &sig.CreatedAt); err != nil {
			return nil, err
		}
		sig.Inputs, err = scanMap(inputs)
		if err != nil {
			return nil, err
		}
		sig.EvidenceIDs, err = scanStrings(ev)
		if err != nil {
			return nil, err
		}
		out = append(out, sig)
	}
	return out, rows.Err()
}

func (sc *Scope) InsertEvidence(ctx context.Context, e domain.Evidence) error {
	payload, err := toJSON(e.Payload)
	if err != nil {
		return err
	}
	prov, err := toJSON(e.Provenance)
	if err != nil {
		return err
	}
	_, err = sc.Tx.Exec(ctx, `
		INSERT INTO evidence (
		  id, tenant_id, case_id, stable_key, version, kind, source, classification,
		  content_hash, payload, provenance, retention_until)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (tenant_id, stable_key) DO NOTHING`,
		e.ID, sc.TenantID, e.CaseID, e.StableKey, e.Version, e.Kind, e.Source, e.Classification,
		e.ContentHash, payload, prov, e.RetentionUntil)
	return err
}

func (sc *Scope) ListEvidence(ctx context.Context, caseID string) ([]domain.Evidence, error) {
	rows, err := sc.Tx.Query(ctx, `
		SELECT id, tenant_id, case_id, stable_key, version, kind, source, classification,
		       content_hash, payload, provenance, created_at, retention_until
		FROM evidence WHERE tenant_id = $1 AND case_id = $2
		ORDER BY created_at ASC, id ASC`, sc.TenantID, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Evidence
	for rows.Next() {
		e, err := scanEvidence(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func scanEvidence(row pgx.Row) (domain.Evidence, error) {
	var e domain.Evidence
	var payload, prov []byte
	err := row.Scan(&e.ID, &e.TenantID, &e.CaseID, &e.StableKey, &e.Version, &e.Kind, &e.Source,
		&e.Classification, &e.ContentHash, &payload, &prov, &e.CreatedAt, &e.RetentionUntil)
	if err != nil {
		return domain.Evidence{}, err
	}
	e.Payload, err = scanMap(payload)
	if err != nil {
		return domain.Evidence{}, err
	}
	e.Provenance, err = scanMap(prov)
	return e, err
}

func (sc *Scope) InsertReport(ctx context.Context, r domain.AIReport) error {
	body, err := toJSON(r.Report)
	if err != nil {
		return err
	}
	_, err = sc.Tx.Exec(ctx, `
		INSERT INTO ai_reports (
		  id, tenant_id, case_id, status, validation_status, provider, model, prompt_version,
		  live, latency_ms, input_tokens, output_tokens, report)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		r.ID, sc.TenantID, r.CaseID, r.Status, r.ValidationStatus, r.Provider, r.Model, r.PromptVersion,
		r.Live, r.LatencyMS, r.InputTokens, r.OutputTokens, body)
	return err
}

func (sc *Scope) ListReports(ctx context.Context, caseID string) ([]domain.AIReport, error) {
	rows, err := sc.Tx.Query(ctx, `
		SELECT id, tenant_id, case_id, status, validation_status, provider, model, prompt_version,
		       live, latency_ms, input_tokens, output_tokens, report, created_at
		FROM ai_reports WHERE tenant_id = $1 AND case_id = $2
		ORDER BY created_at DESC`, sc.TenantID, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.AIReport
	for rows.Next() {
		r, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func scanReport(row pgx.Row) (domain.AIReport, error) {
	var r domain.AIReport
	var body []byte
	err := row.Scan(&r.ID, &r.TenantID, &r.CaseID, &r.Status, &r.ValidationStatus, &r.Provider, &r.Model,
		&r.PromptVersion, &r.Live, &r.LatencyMS, &r.InputTokens, &r.OutputTokens, &body, &r.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AIReport{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.AIReport{}, err
	}
	r.Report, err = scanMap(body)
	return r, err
}

func (sc *Scope) InsertNote(ctx context.Context, n domain.Note) error {
	_, err := sc.Tx.Exec(ctx, `INSERT INTO case_notes (id, tenant_id, case_id, author_id, body) VALUES ($1,$2,$3,$4,$5)`,
		n.ID, sc.TenantID, n.CaseID, n.AuthorID, n.Body)
	return err
}

func (sc *Scope) ListNotes(ctx context.Context, caseID string) ([]domain.Note, error) {
	rows, err := sc.Tx.Query(ctx, `SELECT id, tenant_id, case_id, author_id, body, created_at FROM case_notes WHERE tenant_id = $1 AND case_id = $2 ORDER BY created_at ASC`, sc.TenantID, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Note
	for rows.Next() {
		var n domain.Note
		if err := rows.Scan(&n.ID, &n.TenantID, &n.CaseID, &n.AuthorID, &n.Body, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (sc *Scope) InsertDecision(ctx context.Context, id, caseID, actor, action, from, to, rationale string) error {
	_, err := sc.Tx.Exec(ctx, `
		INSERT INTO case_decisions (id, tenant_id, case_id, actor_id, action, from_status, to_status, rationale)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, id, sc.TenantID, caseID, actor, action, from, to, rationale)
	return err
}

func (sc *Scope) InsertAudit(ctx context.Context, a domain.AuditEvent) error {
	meta, err := toJSON(a.Metadata)
	if err != nil {
		return err
	}
	var actor any
	if a.ActorID != "" {
		actor = a.ActorID
	}
	_, err = sc.Tx.Exec(ctx, `
		INSERT INTO audit_events (id, tenant_id, actor_id, actor_label, action, entity_type, entity_id, rationale, metadata)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		a.ID, sc.TenantID, actor, a.ActorLabel, a.Action, a.EntityType, a.EntityID, a.Rationale, meta)
	return err
}

func (sc *Scope) ListAudit(ctx context.Context, entityType, entityID string) ([]domain.AuditEvent, error) {
	rows, err := sc.Tx.Query(ctx, `
		SELECT id, tenant_id, actor_id, actor_label, action, entity_type, entity_id, rationale, metadata, created_at
		FROM audit_events
		WHERE tenant_id = $1 AND entity_type = $2 AND entity_id = $3
		ORDER BY created_at ASC`, sc.TenantID, entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.AuditEvent
	for rows.Next() {
		var a domain.AuditEvent
		var actor *string
		var meta []byte
		if err := rows.Scan(&a.ID, &a.TenantID, &actor, &a.ActorLabel, &a.Action, &a.EntityType, &a.EntityID, &a.Rationale, &meta, &a.CreatedAt); err != nil {
			return nil, err
		}
		if actor != nil {
			a.ActorID = *actor
		}
		a.Metadata, err = scanMap(meta)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (sc *Scope) CaseTransactions(ctx context.Context, caseID string) ([]domain.Transaction, error) {
	rows, err := sc.Tx.Query(ctx, txnSelect+`
		JOIN case_transactions ct ON ct.transaction_id = t.id AND ct.tenant_id = t.tenant_id
		WHERE t.tenant_id = $1 AND ct.case_id = $2
		ORDER BY t.occurred_at ASC, t.id ASC`, sc.TenantID, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Transaction
	for rows.Next() {
		t, err := scanTxn(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (sc *Scope) Counts(ctx context.Context) (map[string]int, error) {
	rows, err := sc.Tx.Query(ctx, `SELECT status, count(*) FROM cases WHERE tenant_id = $1 GROUP BY status`, sc.TenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var status string
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			return nil, err
		}
		out[status] = n
	}
	return out, rows.Err()
}

func (sc *Scope) InsertJob(ctx context.Context, j domain.Job) (string, error) {
	payload, err := toJSON(j.Payload)
	if err != nil {
		return "", err
	}
	var id string
	err = sc.Tx.QueryRow(ctx, `
		INSERT INTO jobs (id, tenant_id, job_type, idempotency_key, payload, status, max_attempts)
		VALUES ($1, $2, $3, $4, $5, 'pending', $6)
		ON CONFLICT (tenant_id, idempotency_key) DO UPDATE SET idempotency_key = EXCLUDED.idempotency_key
		RETURNING id`, j.ID, sc.TenantID, j.Type, j.IdempotencyKey, payload, j.MaxAttempts).Scan(&id)
	return id, err
}

func (s *Store) RecoverStaleJobs(ctx context.Context) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE jobs SET status = 'pending', locked_at = NULL, updated_at = now()
		WHERE status = 'processing' AND locked_at < now() - interval '2 minutes'`)
	return err
}

func (s *Store) ClaimJob(ctx context.Context) (domain.Job, bool, error) {
	if err := s.RecoverStaleJobs(ctx); err != nil {
		return domain.Job{}, false, err
	}
	row := s.Pool.QueryRow(ctx, `
		WITH next AS (
		  SELECT id FROM jobs
		  WHERE status = 'pending' AND next_attempt_at <= now()
		  ORDER BY created_at
		  FOR UPDATE SKIP LOCKED
		  LIMIT 1
		)
		UPDATE jobs j SET status = 'processing', locked_at = now(), updated_at = now()
		FROM next WHERE j.id = next.id
		RETURNING j.id, j.tenant_id, j.job_type, j.idempotency_key, j.payload, j.status,
		          j.attempts, j.max_attempts, j.next_attempt_at, j.last_error, j.created_at, j.updated_at`)
	j, err := scanJob(row)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.Job{}, false, nil
	}
	if err != nil {
		return domain.Job{}, false, err
	}
	return j, true, nil
}

func scanJob(row pgx.Row) (domain.Job, error) {
	var j domain.Job
	var payload []byte
	err := row.Scan(&j.ID, &j.TenantID, &j.Type, &j.IdempotencyKey, &payload, &j.Status,
		&j.Attempts, &j.MaxAttempts, &j.NextAttemptAt, &j.LastError, &j.CreatedAt, &j.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Job{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Job{}, err
	}
	j.Payload, err = scanMap(payload)
	return j, err
}

func (s *Store) FinishJob(ctx context.Context, id, status, lastError string, attempts int, next time.Time) error {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE jobs SET status = $2, last_error = $3, attempts = $4, next_attempt_at = $5,
		  locked_at = NULL, updated_at = now()
		WHERE id = $1`, id, status, lastError, attempts, next)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) ReplayJob(ctx context.Context, tenantID, id string) error {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE jobs SET status = 'pending', attempts = 0, last_error = '', next_attempt_at = now(),
		  locked_at = NULL, updated_at = now()
		WHERE id = $1 AND tenant_id = $2 AND status = 'dead'`, id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) QueueDepth(ctx context.Context, tenantID string) (int, error) {
	var n int
	err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE tenant_id = $1 AND status IN ('pending','processing')`, tenantID).Scan(&n)
	return n, err
}

func (sc *Scope) InsertAssistant(ctx context.Context, m domain.AssistantMessage) error {
	cites, err := toJSON(m.Citations)
	if err != nil {
		return err
	}
	var author any
	if m.AuthorID != "" {
		author = m.AuthorID
	}
	_, err = sc.Tx.Exec(ctx, `
		INSERT INTO assistant_messages (id, tenant_id, case_id, author_id, role, content, citations, refused, retain_until)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		m.ID, sc.TenantID, m.CaseID, author, m.Role, m.Content, cites, m.Refused, m.RetainUntil)
	return err
}

func (sc *Scope) ListAssistant(ctx context.Context, caseID string, now time.Time) ([]domain.AssistantMessage, error) {
	rows, err := sc.Tx.Query(ctx, `
		SELECT id, tenant_id, case_id, author_id, role, content, citations, refused, created_at, retain_until
		FROM assistant_messages
		WHERE tenant_id = $1 AND case_id = $2 AND retain_until > $3
		ORDER BY created_at ASC`, sc.TenantID, caseID, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.AssistantMessage
	for rows.Next() {
		var m domain.AssistantMessage
		var author *string
		var cites []byte
		if err := rows.Scan(&m.ID, &m.TenantID, &m.CaseID, &author, &m.Role, &m.Content, &cites, &m.Refused, &m.CreatedAt, &m.RetainUntil); err != nil {
			return nil, err
		}
		if author != nil {
			m.AuthorID = *author
		}
		m.Citations, err = scanStrings(cites)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (sc *Scope) PurgeAssistant(ctx context.Context, now time.Time) (int64, error) {
	tag, err := sc.Tx.Exec(ctx, `DELETE FROM assistant_messages WHERE tenant_id = $1 AND retain_until < $2`, sc.TenantID, now)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// CountAccountsBypass is used by the RLS test. It does not set app.tenant_id.
func (s *Store) CountAccountsUnscoped(ctx context.Context) (int, error) {
	conn, err := s.Pool.Acquire(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, `SELECT set_config('app.tenant_id', '', false)`); err != nil {
		return 0, err
	}
	var n int
	err = conn.QueryRow(ctx, `SELECT count(*) FROM accounts`).Scan(&n)
	return n, err
}

func TruncateBusiness(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		TRUNCATE TABLE
		  assistant_messages, jobs, audit_events, case_decisions, case_notes, ai_reports,
		  risk_signals, evidence, case_transactions, cases, transactions, raw_events,
		  locations, devices, merchants, accounts
		RESTART IDENTITY CASCADE`)
	if err != nil {
		return fmt.Errorf("truncate: %w", err)
	}
	return nil
}
