-- FinGuard AI canonical schema. Money is integer minor units. Timestamps are timestamptz (UTC).
-- Tenant isolation is enforced by repository predicates and FORCE ROW LEVEL SECURITY.

CREATE TABLE tenants (
  id text PRIMARY KEY,
  name text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE users (
  id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES tenants(id),
  email text NOT NULL UNIQUE,
  display_name text NOT NULL,
  role text NOT NULL CHECK (role IN ('viewer', 'investigator', 'senior_investigator', 'admin')),
  password_hash text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
  id text PRIMARY KEY,
  user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash text NOT NULL UNIQUE,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX sessions_expires_idx ON sessions (expires_at);

CREATE TABLE accounts (
  id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES tenants(id),
  external_id text NOT NULL,
  display_label text NOT NULL,
  currency text NOT NULL,
  status text NOT NULL DEFAULT 'open',
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, external_id)
);

CREATE TABLE merchants (
  id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES tenants(id),
  external_id text NOT NULL,
  name text NOT NULL,
  category text NOT NULL DEFAULT '',
  country text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, external_id)
);

CREATE TABLE devices (
  id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES tenants(id),
  external_id text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, external_id)
);

CREATE TABLE locations (
  id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES tenants(id),
  country text NOT NULL DEFAULT '',
  region text NOT NULL DEFAULT '',
  city text NOT NULL DEFAULT '',
  natural_key text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, natural_key)
);

CREATE TABLE raw_events (
  id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES tenants(id),
  idempotency_key text NOT NULL,
  source_system text NOT NULL,
  source_id text NOT NULL DEFAULT '',
  schema_version text NOT NULL,
  payload jsonb NOT NULL,
  payload_hash text NOT NULL,
  status text NOT NULL CHECK (status IN ('accepted', 'quarantined')),
  quarantine_reason text NOT NULL DEFAULT '',
  received_at timestamptz NOT NULL DEFAULT now(),
  retention_until timestamptz NOT NULL,
  UNIQUE (tenant_id, idempotency_key)
);
CREATE INDEX raw_events_tenant_received_idx ON raw_events (tenant_id, received_at DESC);

CREATE TABLE transactions (
  id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES tenants(id),
  raw_event_id text NOT NULL REFERENCES raw_events(id),
  source_system text NOT NULL,
  source_id text NOT NULL,
  account_id text NOT NULL REFERENCES accounts(id),
  merchant_id text NOT NULL DEFAULT '',
  device_id text NOT NULL DEFAULT '',
  location_id text NOT NULL DEFAULT '',
  location_country text NOT NULL DEFAULT '',
  amount_minor bigint NOT NULL CHECK (amount_minor >= 0),
  currency text NOT NULL,
  status text NOT NULL CHECK (status IN ('approved', 'declined', 'pending')),
  occurred_at timestamptz NOT NULL,
  memo text NOT NULL DEFAULT '',
  schema_version text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, source_system, source_id)
);
CREATE INDEX transactions_account_time_idx ON transactions (tenant_id, account_id, occurred_at);
CREATE INDEX transactions_device_idx ON transactions (tenant_id, device_id);

CREATE TABLE cases (
  id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES tenants(id),
  account_id text NOT NULL REFERENCES accounts(id),
  status text NOT NULL CHECK (status IN ('open', 'in_review', 'escalated', 'resolved', 'closed')),
  priority text NOT NULL CHECK (priority IN ('low', 'medium', 'high')),
  title text NOT NULL,
  assignee_id text REFERENCES users(id),
  version integer NOT NULL DEFAULT 1 CHECK (version > 0),
  signal_count integer NOT NULL DEFAULT 0,
  opened_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  resolved_at timestamptz,
  resolution text NOT NULL DEFAULT '',
  rationale text NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX cases_one_active_per_account
  ON cases (tenant_id, account_id)
  WHERE status IN ('open', 'in_review', 'escalated');
CREATE INDEX cases_tenant_opened_idx ON cases (tenant_id, opened_at DESC, id DESC);

CREATE TABLE case_transactions (
  tenant_id text NOT NULL REFERENCES tenants(id),
  case_id text NOT NULL REFERENCES cases(id),
  transaction_id text NOT NULL REFERENCES transactions(id),
  linked_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (case_id, transaction_id)
);

CREATE TABLE evidence (
  id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES tenants(id),
  case_id text NOT NULL REFERENCES cases(id),
  stable_key text NOT NULL,
  version integer NOT NULL DEFAULT 1,
  kind text NOT NULL,
  source text NOT NULL,
  classification text NOT NULL,
  content_hash text NOT NULL,
  payload jsonb NOT NULL,
  provenance jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  retention_until timestamptz NOT NULL,
  UNIQUE (tenant_id, stable_key)
);
CREATE INDEX evidence_case_idx ON evidence (tenant_id, case_id, created_at);

CREATE TABLE risk_signals (
  id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES tenants(id),
  case_id text NOT NULL REFERENCES cases(id),
  transaction_id text NOT NULL REFERENCES transactions(id),
  rule_code text NOT NULL,
  rule_version text NOT NULL,
  explanation text NOT NULL,
  threshold text NOT NULL,
  observed_value text NOT NULL,
  inputs jsonb NOT NULL,
  evidence_ids jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, transaction_id, rule_code, rule_version)
);
CREATE INDEX risk_signals_case_idx ON risk_signals (tenant_id, case_id, created_at);

CREATE TABLE ai_reports (
  id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES tenants(id),
  case_id text NOT NULL REFERENCES cases(id),
  status text NOT NULL CHECK (status IN ('ready', 'rejected', 'failed')),
  validation_status text NOT NULL,
  provider text NOT NULL,
  model text NOT NULL,
  prompt_version text NOT NULL,
  live boolean NOT NULL,
  latency_ms integer NOT NULL DEFAULT 0,
  input_tokens integer NOT NULL DEFAULT 0,
  output_tokens integer NOT NULL DEFAULT 0,
  report jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ai_reports_case_idx ON ai_reports (tenant_id, case_id, created_at DESC);

CREATE TABLE case_notes (
  id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES tenants(id),
  case_id text NOT NULL REFERENCES cases(id),
  author_id text NOT NULL REFERENCES users(id),
  body text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE case_decisions (
  id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES tenants(id),
  case_id text NOT NULL REFERENCES cases(id),
  actor_id text NOT NULL REFERENCES users(id),
  action text NOT NULL,
  from_status text NOT NULL,
  to_status text NOT NULL,
  rationale text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE audit_events (
  id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES tenants(id),
  actor_id text,
  actor_label text NOT NULL,
  action text NOT NULL,
  entity_type text NOT NULL,
  entity_id text NOT NULL,
  rationale text NOT NULL DEFAULT '',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_entity_idx ON audit_events (tenant_id, entity_type, entity_id, created_at);

CREATE TABLE jobs (
  id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES tenants(id),
  job_type text NOT NULL,
  idempotency_key text NOT NULL,
  payload jsonb NOT NULL,
  status text NOT NULL CHECK (status IN ('pending', 'processing', 'done', 'dead')),
  attempts integer NOT NULL DEFAULT 0,
  max_attempts integer NOT NULL DEFAULT 5,
  next_attempt_at timestamptz NOT NULL DEFAULT now(),
  locked_at timestamptz,
  last_error text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, idempotency_key)
);
CREATE INDEX jobs_claim_idx ON jobs (status, next_attempt_at);

CREATE TABLE assistant_messages (
  id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES tenants(id),
  case_id text NOT NULL REFERENCES cases(id),
  author_id text,
  role text NOT NULL CHECK (role IN ('investigator', 'assistant', 'system')),
  content text NOT NULL,
  citations jsonb NOT NULL DEFAULT '[]'::jsonb,
  refused boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  retain_until timestamptz NOT NULL
);
CREATE INDEX assistant_case_idx ON assistant_messages (tenant_id, case_id, created_at);

CREATE OR REPLACE FUNCTION finguard_forbid_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'append-only table % cannot be updated or deleted', TG_TABLE_NAME;
END;
$$;

CREATE TRIGGER audit_no_mutation
  BEFORE UPDATE OR DELETE ON audit_events
  FOR EACH ROW EXECUTE FUNCTION finguard_forbid_mutation();

CREATE TRIGGER evidence_no_mutation
  BEFORE UPDATE OR DELETE ON evidence
  FOR EACH ROW EXECUTE FUNCTION finguard_forbid_mutation();

-- Row level security. Superusers bypass this; the application connects as finguard_app.
DO $$
DECLARE
  tbl text;
BEGIN
  FOREACH tbl IN ARRAY ARRAY[
    'accounts','merchants','devices','locations','raw_events','transactions','cases',
    'case_transactions','evidence','risk_signals','ai_reports','case_notes','case_decisions',
    'audit_events','assistant_messages'
  ]
  LOOP
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', tbl);
    EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', tbl);
    EXECUTE format($p$
      CREATE POLICY tenant_isolation ON %I
      USING (tenant_id = current_setting('app.tenant_id', true))
      WITH CHECK (tenant_id = current_setting('app.tenant_id', true))
    $p$, tbl);
  END LOOP;
END $$;

GRANT USAGE ON SCHEMA public TO finguard_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON
  tenants, users, sessions, accounts, merchants, devices, locations, raw_events,
  transactions, cases, case_transactions, risk_signals, ai_reports, case_notes,
  case_decisions, jobs, assistant_messages
TO finguard_app;
GRANT SELECT, INSERT ON evidence, audit_events TO finguard_app;
