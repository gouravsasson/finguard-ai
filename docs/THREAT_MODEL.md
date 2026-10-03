# Threat model

FinGuard is a decision-support demonstrator. The attacker goals that matter here are cross-tenant reads, forged AI findings, prompt injection that causes a side effect, and stolen demo credentials used as if they were production secrets.

## Assets

Case evidence, session tokens, the Gemini key, and the audit trail.

## Trust boundaries

1. Browser to API. The UI does not call Gemini.
2. API to PostgreSQL. The app role is subject to row-level security.
3. Worker to Gemini. The adapter attaches no tools.
4. Tenant to tenant. `tenant_id` comes from the session.
5. External event text to the rule engine. Memos are not rule inputs.

## STRIDE notes

| Threat | Control |
| --- | --- |
| Spoofing | HttpOnly session cookie, bcrypt passwords, no public signup |
| Tampering | Append-only audit and evidence triggers; optimistic case version |
| Repudiation | Human actions store actor, time, and rationale |
| Information disclosure | RLS and `tenant_id` predicates; redaction; logs omit bodies |
| Denial of service | Body limit, batch cap of 100, rate limit, AI daily cap, timeouts |
| Elevation | Role checks on ingest, notes, reports, escalation, retention |

## Prompt injection

Merchant names and memos are untrusted data inside a delimited JSON block. The system prompt tells the model to ignore instructions there. The process never gives the model a tool. Questions that ask to freeze, transfer, or accuse are refused before a model call. Output that includes a tool-call field is rejected.

## Residual risk

- Local demo passwords and the `finguard_app` password are intentionally obvious and must not be reused in production. The API refuses those defaults when `ENV=production`.
- Rate limits and the AI cap are per process.
- A superuser database connection bypasses RLS. Production should not use one.
- This document is not a penetration test.
