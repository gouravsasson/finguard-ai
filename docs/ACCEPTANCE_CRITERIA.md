# Acceptance criteria

| Area | Check | Evidence | Severity |
| --- | --- | --- | --- |
| Repository | Setup is documented | README commands | Blocker |
| Build | Go tests compile and pass; frontend `tsc` passes | Local test run on 2026-10-03 | Blocker |
| Migrations | Fresh migrate and rollback | `TestMigrationRollback` | Blocker |
| Tenant isolation | Tenant B cannot read tenant A | `TestDemoPath` 404 and unscoped account count 0 | Blocker |
| Authentication | Viewer and investigator denied for senior actions | `TestDemoPath` 403s | Blocker |
| Idempotency | Replay and conflicting payload | `TestDemoPath` replayed and 409 | Blocker |
| Risk rules | Same rules, evidence ids, catalog match | `TestSeedCatalog`, `internal/rules` | High |
| Evidence | Report facts cite ids in the case | `TestDemoPath` | Blocker |
| AI safety | Freeze question refused; memo is not a rule input | `TestDemoPath`, `TestPromptIsolatesMemo` | Blocker |
| AI fallback | Stub is `live: false`; failures leave a case | Report service and UI status | High |
| Audit | Resolve stores actor and rationale | `TestDemoPath` | Blocker |
| Graph | Edges are observed; shared device is not identity | `TestSharedDeviceIsObservedNotIdentity` | High |
| Privacy | Secret scan clean; redaction test | `scripts/secret-scan.sh`, `redact` tests | Blocker |
| Performance | Local sequential ingest timing only | 1–9 ms per event in one demo-path log | High |
| Operations | Health, readiness, runbooks | `/healthz`, `/readyz`, `docs/ops` | High |
| Demo | Seeded path from ingest to resolution | `TestSeedCatalog` and the UI | High |
| Claims | No accuracy or compliance claim | README and limitations | Blocker |
