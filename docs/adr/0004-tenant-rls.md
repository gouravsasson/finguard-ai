# ADR 0004 — Tenant filters and row-level security

Every business query includes `tenant_id` from the session. The app role also sets `app.tenant_id` and the tables force row-level security. Superusers bypass RLS, so production traffic must not use a superuser. Tests check that an unscoped count as `finguard_app` returns zero.

Status: accepted.
