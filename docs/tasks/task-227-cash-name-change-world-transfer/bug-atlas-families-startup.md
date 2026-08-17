# atlas-families: image name confusion, then a fatal migration

Two separate problems surfaced when this branch's `deploy/k8s/base/atlas-families.yaml`
made atlas-families run for the first time. Neither originates in task-227's own
scope; both were latent until the service was actually deployed.

## 1. The image name is NOT a mismatch (no code change made)

The GHCR package `ghcr.io/chronicle20/atlas-family/atlas-family` (singular) is the
**legacy standalone-repo** package: it carries only `latest`, `latest-amd64`,
`latest-arm64`, with no `main-<sha>` tags. Every image the monorepo pipeline
publishes carries `main-<sha>-amd64/arm64`.

`atlas-families/atlas-families` (plural) is the monorepo package and is correct:

- `.github/config/services.json` on `origin/main` already declares
  `"docker_image": "ghcr.io/chronicle20/atlas-families/atlas-families"`, predating
  this branch.
- All 66 services use the identical `<name>/<name>` pattern — zero deviations.
- main-publish run 32020103573 (commit `16a6c1c`, success) built atlas-families
  under that name, and the image-bump bot then pinned `newTag: main-16a6c1c` in the
  main overlay on `main`.

The 403 on the plural name was **package visibility**: new GHCR packages default to
private, and `main-publish.yml` only requests `packages: write` — it never sets
visibility, so the older packages must have been made public by hand. The service
deployments carry no `imagePullSecrets`, so a private package means
`ImagePullBackOff`. Resolved by the maintainer making the package public.

Nothing in the repo references the singular name (the only `atlas-family` hits are
Go import paths — `go.mod` declares the module as `atlas-family` — which the
Dockerfile never uses; it discovers the module dir via
`ls -d services/${SERVICE}/atlas.com/*/`). Renaming to the singular form would point
the deployment at the stale legacy image.

**If atlas-families is not the last new service, the same private-by-default trap
will hit the next one.** Worth either adding a visibility step to `main-publish.yml`
or making it part of the new-service checklist in `docs/adding-a-new-service.md`.

## 2. Fatal migration — fixed in commit `7b4ef517e`

Once the image pulled, the pod crash-looped:

```
ERROR: function array_length(text, integer) does not exist (SQLSTATE 42883)
fatal  Migrating schema.
```

`Entity.JuniorIds` uses `gorm:"serializer:json"`, so `junior_ids` is a **TEXT**
column holding a JSON array (`"[1,2]"`) — confirmed by introspection,
`data_type = "text"` — but `check_junior_count` was written with `array_length()`,
a Postgres ARRAY function. `Migration()` runs before the service serves anything, so
the rejected `ALTER TABLE` aborted startup.

Fix (`family/entity.go`):

```sql
CHECK (junior_ids IS NULL OR json_array_length(junior_ids::json) <= 2)
```

The NULL allowance is preserved — a nil slice serializes to SQL NULL, an empty slice
to `"[]"` (counts as 0).

**Why the existing suite missed it:** `Migration()`'s constraint block is guarded by
`if dialectName == "postgres"`, and the module's tests run on SQLite, so the
constraint SQL was never parsed. `family/entity_migration_test.go` (new) runs
`Migration()` against a real Postgres container using the `tcpostgres` pattern
already established in `atlas-character/pending_change/entity_test.go`. Verified the
new test reproduces the exact production error when the `array_length()` expression
is restored, and that the rewritten constraint still rejects a third junior
(SQLSTATE 23514) while accepting NULL, `"[]"`, and two.

This adds `testcontainers-go/modules/postgres` to the families module (additive
go.mod/go.sum only — `go mod tidy` was deliberately not run, as it re-classified
several unrelated direct deps as indirect).

## Next step

Redeploy and confirm the pod reaches Ready. Any further startup failure is a
different defect — this one is verified against real Postgres 16, not inferred.
