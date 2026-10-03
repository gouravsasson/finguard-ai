# Implementation plan

The build follows the playbook phases in one repository. Each phase below names the outcome that is in the tree now.

| Phase | Outcome |
| --- | --- |
| 00 | This assessment and plan. No preexisting product code to preserve. |
| 01 | Product requirements, threat model, acceptance criteria. |
| 02 | Architecture notes and ADRs. |
| 03 | `migrations/001_init.sql` and domain types. |
| 04 | `/api/v1` routes, OpenAPI, request ids, error envelope, health checks. |
| 05 | Session cookie, roles, tenant from the session, RLS plus query filters. |
| 06 | Authenticated ingest, idempotency, quarantine, import CLI. |
| 07 | Version `2026.1` rules. No model scoring. |
| 08 | Cases, state machine, transactional outbox, worker backoff. |
| 09 | Immutable evidence ids and bounded packages. |
| 10 | Provider interface, Gemini adapter, stub, timeout and retry. |
| 11 | Evidence-id validation and tool-request rejection. |
| 12 | Case-scoped assistant that refuses side effects. |
| 13 | Observed graph. Shared device is not an identity edge. |
| 14 | Notes, assignment, escalation, resolution, audit. |
| 15 | Investigator UI with evidence beside claims. |
| 16 | `finguard seed` and the scenario catalog. |
| 17 | Unit, contract, and integration tests, including migration rollback. |
| 18 | Redaction, headers, rate limit, secret scan, security note. |
| 19 | Dockerfile, Compose, CI workflow, Terraform sketch, runbooks. |
| 20 | README, demo script, limitations, judge FAQ. |

## Commands run on 2026-10-03

```sh
go test ./internal/rules ./internal/aigateway ./internal/api ./internal/auth ./internal/graph ./internal/ingestion ./internal/jobs ./internal/redact
go test ./internal/integration -count=1 -v
cd frontend && npm test && npx tsc --noEmit
sh scripts/secret-scan.sh
go run ./cmd/finguard seed --reset
```

Integration result: `TestDemoPath` pass (0.16s), `TestSeedCatalog` pass (0.31s), `TestMigrationRollback` pass (0.14s). In that demo-path run, individual event posts on local Docker Postgres were about 1–9 ms. That is one sequential test on one laptop, not a capacity claim.

## Not done

- No Cloud Run URL. Terraform has not been applied.
- No live Gemini call in this environment. `GEMINI_API_KEY` was not set. The stub path was tested and is labeled `live: false`.
- `govulncheck` was not installed in this environment at build time.
