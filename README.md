# FinGuard AI

FinGuard is a local, synthetic-data demonstrator for evidence-grounded financial fraud **investigation**. Deterministic rules create explainable signals. An AI draft, when requested, may only cite evidence supplied for that case. A person decides what happens next.

It does not move money, freeze accounts, accuse anyone, or claim measured fraud-detection accuracy.

## Run the demo

Requirements: Docker, Go 1.26, Node 22.

```sh
docker compose up -d postgres
export DATABASE_URL='postgres://finguard_app:finguard_app@localhost:54329/finguard?sslmode=disable'
export MIGRATE_DATABASE_URL='postgres://finguard:finguard@localhost:54329/finguard?sslmode=disable'
export AI_PROVIDER=stub
go run ./cmd/finguard seed --reset
go run ./cmd/finguard api
```

In another terminal:

```sh
cd frontend && npm install && npm run dev
```

Open http://localhost:5173 and sign in as `senior@northwind.demo` / `synthetic-demo-pass`.

Other synthetic people, same password: `investigator@northwind.demo`, `viewer@northwind.demo`, `admin@northwind.demo`, `investigator@southbank.demo`.

The three-minute path is in [docs/demo/DEMO_SCRIPT.md](docs/demo/DEMO_SCRIPT.md). Start with the case titled for account `acct_demo`.

## What “live model” means

`AI_PROVIDER=auto` calls Gemini only when `GEMINI_API_KEY` is set. Otherwise the local stub writes a grounded draft and the API marks `live: false`. A stub result is not a successful Gemini call. If Gemini is configured and the call fails, the case stays open and the report status is `failed` or `rejected`.

## Tests

```sh
export MIGRATE_DATABASE_URL='postgres://finguard:finguard@localhost:54329/finguard?sslmode=disable'
export TEST_DATABASE_URL='postgres://finguard_app:finguard_app@localhost:54329/finguard_test?sslmode=disable'
go test ./...
cd frontend && npm test && npm run build
```

Integration tests use database `finguard_test` and will replace that database’s business tables. They do not reset the demo database `finguard` unless you run `seed --reset`.

## Layout

- `cmd/finguard` — `api`, `worker`, `migrate`, `rollback`, `seed`, `import`
- `internal/` — ingestion, rules, cases, evidence, AI gateway, graph, jobs
- `migrations/` — PostgreSQL schema
- `frontend/` — investigator desk
- `docs/` — requirements, threat model, architecture, demo, operations

## Honest limits

See [docs/demo/LIMITATIONS.md](docs/demo/LIMITATIONS.md). This repository is a competition-style demonstrator with synthetic records. It is not a production banking system and it is not certified.

## Website and UI verification

The public landing page is at `/`, sign-in is at `/login`, and the investigation queue is at `/workspace`. A protected case link returns to the requested case after sign-in.

The landing photography is stored locally with attribution in `frontend/public/images/ATTRIBUTION.md`. Its product preview is a screenshot of the actual synthetic workspace.

Run `sh scripts/verify-browser.sh` for the browser feature and accessibility checks. This command resets only the dedicated `finguard_test` database and uses port 8081. Install the browser first with `(cd frontend && npx playwright install chromium)`.

See [UI and feature verification](docs/UI_FEATURE_VERIFICATION.md) for executed checks, fixed defects, and verification limits.
