# Repository assessment

Date: 2026-10-03.

## Current state at start

The workspace `/Users/gourav/Desktop/office` is a collection of unrelated projects. There was no FinGuard application, no Go module, no fraud-investigation schema, and no shared frontend to preserve. Sibling directories were left untouched.

FinGuard was created as a new module at `finguard-ai/`. That is a greenfield build, not a rewrite of an existing service.

## Baseline before feature code

There were no existing tests, builds, or CI workflows for this product. The baseline is “empty”. Failures recorded: none, because there was no suite to run.

## Stack chosen

- Go 1.26 API and worker
- PostgreSQL 16, integer money in minor units, `timestamptz`
- React 19 + TypeScript investigator UI
- Gemini through `google.golang.org/genai`, behind a provider interface, with a labeled local stub
- Docker Compose for local Postgres; Terraform sketches Cloud Run, Cloud SQL, and Secret Manager and has not been applied

## Risks

- No GCP project is configured here, so there is no deployed URL.
- Row-level security does not apply to the local superuser. The app connects as `finguard_app`.
- The embedded worker and in-memory rate limit are single-process controls.
- Gemini output is untrusted until the validator accepts its evidence ids.

## Decision

Proceed with a modular monolith and a database-backed job table. Do not add Kafka, Redis, or a second frontend framework.
