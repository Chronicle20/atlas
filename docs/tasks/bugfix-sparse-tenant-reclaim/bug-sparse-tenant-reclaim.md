# Sparse PR teardown never reclaims its tenant

Diagnosed 2026-08-21 from a live investigation that started as "why is
atlas-pr-1441 stuck deleting in Argo". Five defects, all confirmed against the
running cluster, not inferred. Together they mean **no sparse PR teardown has
ever reclaimed its tenant**, and every sparse PR Application wedges in
`Terminating`.

The leaked state from PRs 1411/1412/1437/1441 was reclaimed by hand during the
investigation (115,789 rows across 96 tables, 12 MinIO prefixes, 4 tenant
registry rows, 19 control-plane rows). This branch fixes the causes so it stops
recurring; it does not need to reclaim anything itself.

## Bug 1 — atlas-data panics on every tenant purge (root cause of the wedge)

`services/atlas-data/atlas.com/data/tenantpurge/handler.go:48`

```go
t := tenant.MustFromContext(r.Context())
```

`ParseTenant` (`libs/atlas-rest/server/handler.go:149-158`) builds a *new*
context and passes it as an argument — it never reassigns the request:

```go
tctx := tenant.WithContext(ctx, t)
...
tctx = env.WithContext(tctx, resolved)
next(tl, tctx)(w, r)          // r still carries the ORIGINAL context
```

So `r.Context()` has no tenant, `MustFromContext` panics, and the pod drops the
connection. Observed panic in atlas-main:

```
panic({0x2547220?, 0xc7bc1910860?})
libs/atlas-tenant.MustFromContext(...)  processor.go:84
atlas-data/tenantpurge.purgeInner.func1.1  handler.go:48
...server.RegisterHandler.func1.1.1.1.ParseTenant.2
```

Reproduced directly against a healthy atlas-main (4/4 pods Running):
`DELETE /api/data/tenants/<uuid>` with the operator + tenant headers returns
`curl: (52) Empty reply from server`; through the ingress it surfaces as 502.

This is what `predelete-purge.sh` calls. The hook exits non-zero by design on
failure, so `pre-delete-finalizer.argocd.argoproj.io/cleanup` never clears and
the Application sits in `Terminating` forever. PRs 1411, 1437 and 1441 all
wedged this way.

**Fix:** read the tenant from `d.Context()`, the convention used by 13 other
handlers (`grep -rn "tenant.MustFromContext(d.Context())" services/`).

**Same bug, second site:** `services/atlas-renders/atlas.com/renders/mapr/handler.go:52`
is the only other `r.Context()` caller and panics identically. Fix both.

Add a regression test that exercises the handler through `ParseTenant` rather
than injecting a pre-populated context — the existing
`tenantpurge/handler_test.go` `purgeViaRouter` helper passes because it builds
the context by hand, which is exactly why this shipped.

## Bug 2 — `sweep-orphans.sh --sweep-tenant` targets databases that do not exist

`services/atlas-pr-bootstrap/scripts/tenant-tables.txt` lists bare base names:

```
atlas-accounts accounts
atlas-bans bans
```

The actual databases are `<base>-<envhash>`:

```
atlas-accounts-237a   atlas-accounts-6bb7   atlas-accounts-main
atlas-data-237a       atlas-data-6bb7       atlas-data-main
```

`sweep_tenant` (`sweep-orphans.sh:231`) connects to `-d "$db"` verbatim, so
every one of the 96 deletes fails with `relation does not exist` / unknown
database. It is invisible because psql's stderr goes to `/dev/null` and the
warning is only `delete from <db>.<table> failed`.

**Fix:** resolve the database name against the environment being swept (the
baseline suffix for a sparse tenant, i.e. `<base>-main`). Surface the psql
error in the warning instead of discarding it — a silent 96/96 failure rate is
the reason this went unnoticed.

Also drop `atlas-quest quest_medal_maps` from `tenant-tables.txt`, or regenerate
it: that relation does not exist and errors on every run.

## Bug 3 — `_dcp_reclaim` never lists anything (curl glob)

`services/atlas-pr-bootstrap/scripts/cleanup.sh:231`

```sh
curl -fsS ... "${list_url}?page[size]=250&page[number]=${page}"
```

Without `-g`, curl parses `[size]` as a glob range and exits 3:

```
curl: (3) bad range in position 84:
http://.../api/configurations/services?page[size]=250&page[number]=1
```

So `drop-control-plane` always takes the `list failed` path and deletes
nothing — including the `atlas-tenants` registry row it is responsible for.
Adding `-g` fixed it in the manual reclaim.

**Fix:** add `-g` (or percent-encode the brackets) on every paginated curl in
`cleanup.sh` and `sweep-orphans.sh`. Grep for `page[size]` across the scripts;
do not fix only the one site above.

## Bug 4 — `ATLAS_MODE=sparse` never reaches teardown

`deploy/k8s/overlays/pr-sparse/kustomization.yaml:168-175` patches `ATLAS_MODE`
and `ATLAS_ENVIRONMENT` onto the `atlas-pr-bootstrap` Job only. The teardown
path — the `atlas-pr-cleanup-<N>` Job in `argocd`, whose env is `PR_NUMBER` +
`ATLAS_UI_BASE` + the `atlas-pr-cleanup-env` ConfigMap — never receives it, and
that ConfigMap has no `ATLAS_MODE` key.

`do_sweep_tenant` (`cleanup.sh:325`) therefore evaluates
`${ATLAS_MODE:-isolated}` as `isolated` and skips. Confirmed verbatim in
PR-1412's cleanup log:

```
sweep-tenant  skipped (isolated) — this environment's databases are already dropped by drop-dbs
```

In sparse mode `drop-dbs` drops nothing, so this is an unconditional leak.

**Fix:** make the cleanup Job's mode match the environment it is tearing down.
Prefer deriving it from the live control-plane `environments` record (which
already distinguishes sparse from isolated by existing at all — the same
live-check `_dcp_env_phase` uses) over adding another build-time flag that can
drift. If a flag is used instead, it must be set wherever the cleanup Job is
defined, not only on the bootstrap Job.

## Bug 5 — the image does not ship `tenant-tables.txt`

`services/atlas-pr-bootstrap/Dockerfile` COPYs every other script but not
`scripts/tenant-tables.txt`, so `--sweep-tenant` dies immediately in-cluster:

```
sweep_tenant: missing /atlas/tenant-tables.txt (run tools/gen-tenant-tables.sh)
```

The recovery path documented in the cleanup CronJob's own output
(`sweep-orphans.sh --apply <N>`) is dead as shipped.

**Fix:** add the `COPY` alongside the other `COPY scripts/…` lines. There is a
`test/dockerfile_test.bats` that asserts on image contents — extend it so a
missing data file fails the build rather than a teardown six hours later.

## Related, lower priority — not required for this branch

- `atlas-configurations` `environments` has no DELETE route
  (`services/atlas-configurations/atlas.com/configurations/environments/resource.go:26-30`
  registers GET/POST/GET{name}/PATCH{name} only). Records can only be PATCHed to
  `DELETED`, so the table grows one row per PR forever. Decide whether teardown
  should delete or whether `DELETED` is the intended terminal state.
- The `atlas-pr-cleanup` CronJob only *detects* wedged Applications and prints
  recovery instructions; nothing reaps them. `atlas-pr-1441` would have sat in
  `Terminating` indefinitely.
- `sweep_minio`'s orphan test is "absent from atlas-main", which by construction
  never matches a sparse PR tenant, since sparse tenants live in main's
  registry. Their MinIO prefixes are never reclaimed. The `environments` record
  is the correct ownership marker, not presence in main.

## Verification

`tools/verify.sh` must exit 0 (flagless). Beyond that, the two bugs with live
repros should be verified against behaviour, not just compilation:

- Bug 1: a test that drives the purge handler through the real `ParseTenant`
  middleware and asserts 202, not a panic.
- Bugs 2/3/5: `bats services/atlas-pr-bootstrap/test/` — the suite stubs curl
  and psql, so add cases that assert the *shape* of the emitted command
  (`-g` present, database name suffixed, tenant-tables.txt present in image).
