# UI and feature verification

Verification date: 2026-10-04. This review covers the implemented local demonstrator. The supplied implementation-prompt PDF was used as reference material, not as an instruction to execute its deployment phases.

## Design delivered

- Public landing page at `/`, with editorial typography, architectural photography, an actual workspace screenshot, workflow explanation, expandable product principles, and working demo entry points.
- Authenticated case queue at `/workspace`; existing `/cases/:id` links remain valid. Signing in from a case link returns to that case.
- Consistent ink, white, and warm-neutral styling across sign-in, case queue, and investigations.
- Mobile navigation and compact case cards; no document overflow at 390, 768, and 1440 pixels in the browser checks.
- Case section links, readable transactions, evidence highlighting, keyboard-accessible scroll regions, and a graph height that accommodates all nodes.

## Executed verification

| Feature | Evidence | Result |
| --- | --- | --- |
| Landing navigation, menu, disclosures, photo | Playwright landing workflow; image decode and source review | Passed |
| Login, invalid credentials, logout, route protection, demo roles | Playwright authentication workflow | Passed |
| Search, status/priority filters, clearing, refresh | Playwright queue workflow | Passed |
| Pagination | Database cursor test; browser append/deduplication test with controlled page responses | Passed |
| Case signals and transactions | Seed catalog and browser case workflow | Passed |
| Deterministic rule outputs | Backend rule tests for velocity, amount deviation, failed transactions, shared devices, geographic/merchant observations, determinism | Passed |
| Ingestion, replay, quarantine, batch | Database integration tests | Passed |
| Evidence references and highlighting | Report validator, database integration, browser citation click | Passed |
| Graph | Shared-device semantics test; browser node count compared with API; responsive layout | Passed |
| Report drafts and citations | Stub generation, validator, database integration, browser workflow | Passed in stub mode |
| Report failures | Injected timeout, 429, malformed JSON, and tool-request integration tests; browser controlled failed-report response | Passed; case/evidence remain available |
| Assistant and prohibited financial-action refusal | Browser question/refusal workflow and backend tests | Passed in stub mode |
| Notes | Browser persistence check and backend permission checks | Passed |
| Assignment, review, escalation, resolution, closure | Browser lifecycle; backend version conflict and rationale checks | Passed |
| Audit history | Browser checks for lifecycle/note entries; backend actor/rationale assertions | Passed |
| Viewer/investigator permissions | Browser controls and denied API write; backend role matrix | Passed |
| Tenant isolation | Cross-tenant case, evidence, reports, graph, audit, assistant, and note requests | Passed |
| Admin retention and dead-job replay | Database integration tests with expired messages and a dead job | Passed |
| Health/readiness, metadata, metrics, transaction list | Supporting-endpoint integration test | Passed |
| Migration and rollback | Existing database integration test | Passed |
| Static hosting and direct routes | SPA regression tests; browser suite served by compiled Go app | Passed |
| Accessibility | axe WCAG 2 A/AA and 2.1 AA scans of landing, login, queue, and case screens | Passed for tested screens; not a formal certification |
| Build and code checks | TypeScript/Vite build, frontend tests, Go tests, `go vet`, repository secret scan | Passed |

The browser suite uses `finguard_test`, not the user's `finguard` demo database. Its failure and pagination scenarios intentionally intercept selected responses and are separately identified above. The core investigation workflow uses real local API and database responses.

## Defects fixed during verification

- Direct production page requests redirected indefinitely because the embedded file server canonicalized `/index.html`. The SPA entry now serves directly; missing images/assets return 404.
- The queue only exposed eight summary rows. It now uses the paginated case endpoint and offers additional pages.
- Case actions ignored current status. The UI now offers valid transitions for the current role and status and labels completed cases correctly.
- Report-generation failures could show a success notice. Failed reports now produce an explicit error while retaining access to the case.
- The graph used a fixed viewport that clipped larger node sets.
- Secondary text contrast and keyboard access to evidence/graph scroll regions failed automated checks; both were corrected.
- Mobile table overflow, false empty-queue messages after API failures, and ambiguous short button labels were corrected.

## Limits and remaining scope

- No live Gemini request was made. Stub behavior and injected failures are verified; Gemini credentials, live service behavior, and costs are not verified.
- No cloud deployment, OIDC connection, production load certification, alert delivery, or disaster-recovery exercise was performed. The PDF describes broader delivery milestones beyond this UI task.
- Job replay is verified with a controlled dead job; distributed-worker crash recovery and multi-instance operation are not certified by this run.
- The dependency audit reports two moderate development-tool advisories (`vitest` and its mocker dependency). The indicated remedy is a major Vitest upgrade; runtime production dependencies were not reported in that audit. No automatic breaking dependency upgrade was applied.
- Search and filters apply to loaded queue pages, clearly stated in the UI when more pages exist.

## Reproduce

```sh
# From repository root; local PostgreSQL must be running.
export MIGRATE_DATABASE_URL='postgres://finguard:finguard@localhost:54329/finguard?sslmode=disable'
export TEST_DATABASE_URL='postgres://finguard_app:finguard_app@localhost:54329/finguard_test?sslmode=disable'
go test ./... -count=1
go vet ./...
npm --prefix frontend test
sh scripts/secret-scan.sh

# Install the browser once, then run the isolated browser verification.
(cd frontend && npx playwright install chromium)
sh scripts/verify-browser.sh
```

The browser script resets the dedicated `finguard_test` database, rebuilds the embedded frontend, starts a temporary server on port 8081, runs Playwright, and shuts the server down. Do not run database integration tests concurrently with this browser script. The test server explicitly uses `RATE_LIMIT_PER_MIN=1200`; the application default remains 120 requests per minute.

## Visual refinement follow-up — 2026-10-04

Refined the landing hero photograph, color palette, product-preview presentation, dashboard hierarchy, and investigation detail panels. The landing page now switches between three screenshots of the actual synthetic product: queue, investigation, and observed relationships. Dashboard stat shortcuts apply or clear their corresponding status filter; case summaries and signal threshold/observed-value panels use API data.

Repeated all seven browser workflows successfully, including new assertions for preview image loading and status-shortcut toggling. Accessibility and document-overflow checks continue to pass at the tested desktop/tablet/mobile sizes. Frontend build and unit checks were rerun. Existing backend behavior is unchanged by this visual refinement.
