# Architecture

```text
Browser (React)
    |  cookie session, no Gemini key
    v
Go API  /api/v1
    |  tenant from session, RBAC, validation
    v
PostgreSQL
    |  canonical records, evidence, audit, jobs
    v
Worker
    |  claims jobs, no side-effect tools
    v
Deterministic rules ----> evidence package ----> Gemini or local stub
    |
    v
Validator (evidence ids, no tool requests)
    |
    v
Investigator decision + append-only audit
```

Trust boundaries: browser/API, API/database, worker/model, tenant/tenant.

The model cannot change case state. Case transitions are explicit human actions with a version check.

## Modules

Ingestion, rules, cases, evidence, AI gateway, graph, jobs, audit. The API is a modular monolith. Jobs live in Postgres. Kafka and Redis are not used.

## Failure modes

- Invalid events are quarantined or rejected. A valid replay does not double-apply.
- Gemini timeout, HTTP 429, or invalid JSON retries with backoff, then dead-letters. The case remains readable.
- Tool-like model output is rejected and not retried as if it were a finding.
- Without an API key, the stub runs and is labeled not live.
