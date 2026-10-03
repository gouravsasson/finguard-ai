# Backup and restore

Local demo:

```sh
docker compose exec -T postgres pg_dump -U finguard finguard > backup.sql
docker compose exec -T postgres psql -U finguard -d finguard < backup.sql
```

Cloud SQL, if the Terraform sketch is applied later, uses the instance backup flag in `deploy/terraform/main.tf`. That configuration has not been applied in this environment, so no cloud backup was taken.

Restore is a controlled step. Do not point a restore at a database that still contains another tenant’s real data; this product is synthetic-only.
