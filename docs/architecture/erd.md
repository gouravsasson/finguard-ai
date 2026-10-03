# ERD

```text
tenants 1--* users 1--* sessions
tenants 1--* accounts 1--* transactions *--* cases
transactions 1--* risk_signals *--1 cases
cases 1--* evidence
cases 1--* ai_reports
cases 1--* case_notes
cases 1--* case_decisions
cases 1--* audit_events
cases 1--* assistant_messages
tenants 1--* jobs
raw_events 1--1 transactions
```

Money is `amount_minor bigint`. Times are `timestamptz`. Evidence and audit reject updates and deletes with a trigger. One active case exists per tenant and account (`open`, `in_review`, `escalated`).

Jobs are not under row-level security so a worker can claim across tenants. Payloads contain ids. API reads filter on `tenant_id`.
