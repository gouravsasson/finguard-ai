DROP TABLE IF EXISTS assistant_messages, jobs, audit_events, case_decisions, case_notes,
  ai_reports, risk_signals, evidence, case_transactions, cases, transactions, raw_events,
  locations, devices, merchants, accounts, sessions, users, tenants CASCADE;
DROP FUNCTION IF EXISTS finguard_forbid_mutation();
