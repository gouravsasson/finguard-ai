# Security report

Reviewed against `docs/THREAT_MODEL.md` on 2026-10-03.

## Controls in the tree

- HttpOnly session cookie, CSRF header on API posts, security headers, body limit, login and request rate limits.
- RBAC tests: viewer cannot note; investigator cannot escalate; senior resolve requires a rationale and the current version.
- Cross-tenant read returns 404. Unscoped `SELECT count(*) FROM accounts` as `finguard_app` returned 0 while rows existed.
- Evidence and audit tables reject UPDATE and DELETE.
- Card-like, email, and token-like substrings are redacted before storage.
- `scripts/secret-scan.sh` reported no high-confidence secrets.
- Gemini adapter does not set tools. Assistant refuses freeze and transfer wording.

## Findings

| Item | Severity | Status |
| --- | --- | --- |
| Local database and demo passwords are committed as examples | Expected for the demo | Refused when `ENV=production` |
| `npm audit` reported 2 moderate issues in the frontend tree | Moderate | Not exploited in this app path; not force-upgraded, so the lockfile stays reproducible. Re-check before a real deploy. |
| `govulncheck` binary was not installed here | Info | Not run. CI can add it later. |
| Terraform and the container image were not scanned in a registry | Info | No image was published. |
| Per-process rate limit and AI cap | Info | Documented. Not a global control. |

No committed API key was found. No high-severity code finding from the tests was left open.
