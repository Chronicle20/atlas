# Bug: task-243's dedupe deleted atlas-drops' canonical service-config row

## What happened (production incident, 2026-08-20)

`af2defbc7` (task-243, PR #1435) merged to `main`. On the next
`atlas-configurations` start, `servicesuniq.Migration` deduped the `services`
table down to one row per `(type, environment)` and created
`idx_services_type_env`.

For `(drops-service, main)` there were four rows. The keeper rule
(`resolveGroup`, `servicesuniq/migration.go`) is:

1. the row whose id equals `uuid5(atlasServiceNS, type + "/" + environment)`
2. else the row with the newest `service_history.created_at`
3. else the lowest id lexicographically

No row matched the derived id, so rule 2 decided. `service_history` shows why
that was fatal:

```
service_id                            | type          | environment | created_at
00000000-0000-0000-0000-000000000000  | drops-service | main        | 2026-07-08 11:43:36
3ff23568-b4ef-44c6-b538-6f576a861a6b  | drops-service | main        | 2026-08-19 17:15:09
bc06161e-0604-4cf3-8580-59b8717e4db7  | drops-service | main        | 2026-08-19 17:15:09
```

The migration kept a 2026-08-19 interloper (`1e8b27de-…`) and **deleted
`00000000-…`** — which is not a placeholder but the CANONICAL PINNED ID for
drops-service, recorded in
`services/atlas-pr-bootstrap/canonical/services/drops-service.json` and baked
into `deploy/k8s/base/atlas-drops.yaml:32`.

`atlas-drops` does `configuration.Init(...)(uuid.MustParse(os.Getenv("SERVICE_ID")))`
(`main.go:62`) at startup, got `record not found` → 500, and crash-looped.
Argo `atlas-main` went Degraded on `Deployment "atlas-drops" exceeded its
progress deadline`.

The canonical ids for all three service types:

| service | canonical id | dedupe outcome |
|---|---|---|
| login-service | `e7fb1d7e-47b8-46bd-97dc-867d93530856` | survived (lucky timestamps) |
| channel-service | `e7fb1d7e-47b8-46bd-97dc-867d93530000` | survived (lucky timestamps) |
| drops-service | `00000000-0000-0000-0000-000000000000` | **DELETED** |

login and channel survived only because their canonical rows happened to carry
the newest history. Nothing in the rule protected them.

## Production repair already applied (do NOT redo)

1. `DELETE /api/configurations/services/1e8b27de-…` (204) — interloper removed.
2. `POST /api/configurations/services` with `canonical/services/drops-service.json` (201).
3. That row landed with `environment=''` (see defect 2), so
   `UPDATE services SET environment='main' WHERE id='00000000-…'`.
4. `kubectl rollout restart deployment/atlas-drops` → 2/2 Running, Argo Healthy.

The database is correct now. This task is about the two CODE defects the
incident exposed.

## Defect 1 — `resolveGroup` has no notion of canonical pinned ids

`servicesuniq/migration.go`. Rules 2 and 3 will silently DELETE a row the
system depends on whenever no candidate matches the derived id. The canonical
ids live in `atlas-pr-bootstrap`, which `atlas-configurations` cannot import,
so the migration has no way to recognise them.

`docs/runbooks/sparse-environments.md` §"Pre-flight" already states the intended
behaviour for this case:

> if any group represents a genuine co-resident row rather than a
> non-idempotent-bootstrap artifact … the dedupe rule cannot resolve it
> correctly. **Stop the rollout and re-decide D3** rather than letting the
> migration silently delete a row that was supposed to stay.

The code does not do that — it deletes. Required change: when NO candidate
matches the derived id, do not fall through to newest-history / lowest-id.
Return the existing unresolvable error naming every candidate, so the migration
fails loudly and an operator decides. Silent deletion of an unidentifiable row
is never acceptable; a loud startup failure is recoverable, a deleted canonical
row is not (the unique index forbids re-creating it).

Blast radius is small: `main` has no duplicate groups left, and
`idx_services_type_env` prevents new ones forming, so this path only triggers on
a baseline that still carries legacy duplicates — exactly where a human should
be looking.

## "Defect 2" — WITHDRAWN, this was operator error, not a bug

**The analysis below is wrong and was NOT fixed. It is kept as a record of a
misdiagnosis so nobody re-derives it.**

An empty `environment` is a deliberate, load-bearing contract for ISOLATED
mode, not a defect. `deploy/k8s/overlays/pr/kustomization.yaml:165` sets the
pod's own `ATLAS_ENVIRONMENT=` empty on purpose:

> Isolated mode registers no control-plane environment record, so it must keep
> `env.Self()==""` (FR-1.5).

and `bootstrap.sh:509-513`'s `upsert_service_config` — the default
`ATLAS_MODE=isolated` path — deliberately POSTs with no `ENVIRONMENT` header.
Both legs resolve to `""` by design.

A first attempt (commit `5c6993106`, dropped before the PR) added a
fallback-then-reject chain that returned HTTP 400 when both legs were empty.
Because `bootstrap.sh` runs `set -euo pipefail` and that POST carries no `||`
guard, it would have hard-failed **every isolated-mode PR bootstrap** at the
service-config step. Caught in review before it shipped.

What actually happened during the production repair: the operator (this
session) POSTed without the `ENVIRONMENT` header, which is the caller's job to
send. The row landed unscoped, exactly as designed. The genuine — and much
smaller — sharp edge is that the POST response echoes the submitted model
rather than the stored row, so an unscoped create looks identical to a scoped
one. That is a reporting wart, not a data-integrity defect, and it is not
addressed here.

### Original (incorrect) writeup follows

## Defect 2 — a create with no `ENVIRONMENT` header silently produces an unscoped row

`services/administrator.go:16-24` inserts
`string(env.MustFromContext(ctx))` as `environment`. That value comes from the
`ENVIRONMENT` request header (`bootstrap.sh:59,85` — "the ENVIRONMENT header is
the ONLY way to stamp" it). A POST without the header inserts `environment=''`.

Such a row is invisible to every environment-scoped consumer while still
returning 200 on an unscoped GET — the POST response echoes the submitted
model, not the stored row, so the create looks successful. Observed directly
during the repair above: the restored row read back fine but `atlas-drops`,
which queries `WHERE environment='main'`, still could not see it. The DELETE
for such a row returns 500 rather than 404, which is how it was noticed.

Required change: a created service-config row must never land with an empty
environment. Absent/empty resolved environment → fall back to the service's own
configured environment; if that is also empty, reject the request rather than
inserting an unscoped row.
