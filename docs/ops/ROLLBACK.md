# Rollback

Schema:

```sh
go run ./cmd/finguard rollback
go run ./cmd/finguard migrate
```

`TestMigrationRollback` drops the application tables and migrates them again on `finguard_test`.

Application: redeploy the previous container image. The API process does not migrate when `ENV=production`; run `finguard migrate` as the release step before traffic.

There is no live Cloud Run revision to roll back here.
