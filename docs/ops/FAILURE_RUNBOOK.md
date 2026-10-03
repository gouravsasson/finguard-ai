# Failure runbook

| Symptom | What still works | What to do |
| --- | --- | --- |
| `/readyz` is 503 | `/healthz` | Check `DATABASE_URL` and that Postgres is up. |
| Report status `failed` | Case, signals, evidence, decisions | Read `jobs.last_error`. Replay a dead job with `POST /api/v1/jobs/{id}/replay` as a senior. |
| Report status `rejected` | The case | The model asked for a tool-like field. Do not treat the text as a finding. |
| Duplicate event post | One transaction | Expect `replayed`. A different body with the same key returns 409. |
| Gemini 429 or timeout | Case remains open | Worker backs off up to 5 attempts, then marks the job dead. |
| Wrong tenant | Not found | Confirm the session. Do not accept a tenant header. |

Poison jobs are not retried as successful findings. Stale `processing` jobs older than two minutes return to `pending`.
