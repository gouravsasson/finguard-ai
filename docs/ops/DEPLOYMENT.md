# Deployment

Local path: Docker Compose Postgres, `finguard migrate` (automatic when `ENV` is not production), `finguard seed`, `finguard api`.

Container: `Dockerfile` builds the UI and a static Go binary. The image was not pushed. No registry credentials were available.

Cloud sketch: `deploy/terraform/main.tf` describes Artifact Registry, Secret Manager, a least-privilege runtime service account, Cloud SQL Postgres 16, and Cloud Run. It was not planned or applied (`terraform` is not required to run the demo). Staging and production are separated by the `environment` variable. Secrets are references, not literals.

Production startup must set `ENV=production`, a non-default database password, `COOKIE_SECURE=true`, and a non-default `DEMO_PASSWORD`. The process refuses the local demo secrets in that mode.

CI: `.github/workflows/ci.yml` runs Go tests against Postgres 16 and the frontend test and build. That workflow has not been executed on GitHub from this machine.
