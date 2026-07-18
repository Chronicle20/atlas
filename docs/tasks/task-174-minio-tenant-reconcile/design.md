# task-174 — MinIO tenant-prefix reconciliation + teardown hardening

## Problem

The shared MinIO instance (`minio.minio.svc.cluster.local:9000`, `minio`
namespace) is written by **`atlas-main` and every live PR env** (`atlas-pr-<N>`).
Each env has its own per-namespace `atlas-tenants` and generates its own tenant
UUIDs. Per-tenant WZ/asset/render objects are keyed `tenants/<uuid>/...` in the
`atlas-wz`, `atlas-assets`, and `atlas-renders` buckets. A per-tenant full WZ
ingest alone is ~1.9 GB.

The **only** mechanism that scrubs a torn-down env's objects from this shared
store is the Argo **PreDelete hook** `services/atlas-pr-bootstrap/scripts/predelete-purge.sh`,
which runs once at env teardown, enumerates the *then-live* tenant list, and
issues `DELETE /api/data/tenants/{id}` per tenant. The purge codec itself is
correct — `tenantpurge.Purge` keys off `tenants/<id>/` (matching the write scheme
in `wzinput/scope.go`) and `RemovePrefix` works.

The defect is **invocation coverage**: the hook is best-effort and
enumeration-driven. If it never fires, fails while the namespace is force-deleted,
or the tenant is already deregistered at that instant, the tenant's shared-MinIO
objects leak **permanently** — there is **no reconciliation sweep** that later
compares MinIO `tenants/*` prefixes against the live tenant set. This is the same
class as the known "Ephemeral env DBs leak on teardown" pattern.

### Additional root cause found during implementation (2026-07-17)

Every atlas-data REST route is wrapped by `ParseTenant`
(libs/atlas-rest/server/handler.go), which returns **400** unless all four
headers `TENANT_ID`/`REGION`/`MAJOR_VERSION`/`MINOR_VERSION` are present — even
for operator/cross-tenant routes. The predelete hook's per-tenant
`DELETE /api/data/tenants/{id}` sends **only** `X-Atlas-Operator: 1` and no
tenant headers, so `ParseTenant` 400s it and the purge **never deletes**. This
is very likely *the* reason the incident tenants leaked (not merely an
occasional hook miss). Verified live: an operator atlas-data GET with no tenant
headers → 400; with a synthetic tenant (`00000000-…-000`, `GMS`, `83`, `1`) →
200. Fix: send the synthetic tenant headers on the DELETE (Task 6) and on the
new reconcile POST (Task 4). The reconcile endpoint keeps the standard
`ParseTenant`-wrapped routing (service convention) and its callers supply the
synthetic tenant it accepts-and-ignores.

### Incident that motivated this

- Tenant `1cccd449-6751-4cdd-9b1a-2c33f4b6834d` (not among the 6 live
  `atlas-main` tenants) leaked **~2.34 GB** across all three buckets
  (`atlas-wz` 1.9 GB, `atlas-assets` 434 MB, `atlas-renders` 3.2 MB).
- A second orphan `4f0c070d-5449-47fd-ac67-fe5a98b2a634` leaked into
  `atlas-renders`.
- Both were reclaimed manually (backend `rm -rf` of the `tenants/<id>` subtrees
  on the single-node `xl-single` store); `/data` dropped 19 GB → 17 GB. That
  manual reclaim is the immediate remediation — this task builds the systemic
  backstop so it does not recur.

## Goals

1. **A — reconciliation backstop:** a scheduled sweep that reclaims any
   `tenants/<uuid>/` prefix matching no live tenant across *all* atlas
   namespaces, guarded so it can never delete live or in-flight data.
2. **B — teardown hardening:** make the PreDelete purge hook resilient to
   transient failures so the backstop has to fire less often.

## Non-goals

- Changing the object key scheme or the existing `tenantpurge.Purge` codec.
- Reclaiming `shared/` or `atlas-canonical/baseline/` data — those are not
  per-tenant and are out of scope.
- Per-object ownership tagging at write time (considered as option C; rejected
  to avoid touching the atlas-data write path).

## Design decisions (settled)

| Decision | Choice |
|---|---|
| Scope | A (reconciliation backstop) **+** B (hardened teardown) |
| Split | CronJob orchestrator + atlas-data executor endpoint |
| Rollout | Dry-run first; flip a ConfigMap flag to enable deletion |
| Age guard | 48 h since the prefix's newest object |
| Schedule | Daily |

## Architecture

```
                 (daily CronJob, atlas-main ns)
   ┌───────────────────────────────────────────────────────┐
   │  atlas-minio-reconcile  (ORCHESTRATOR)                 │
   │  1. list ns: atlas-main + atlas-pr-*                   │
   │  2. GET /api/tenants from each env's ingress           │
   │  3. union UUIDs -> keep-list   (FAIL-CLOSED)           │
   │  4. POST keep-list + minAgeHours + dryRun ─────────────┼──► atlas-data
   │  5. log the returned report                            │      (EXECUTOR)
   └───────────────────────────────────────────────────────┘   POST /api/data/minio/reconcile
                                                                 - list tenants/<uuid>/ in 3 buckets
                                                                 - drop uuid in keep-list
                                                                 - drop newest-object age < 48h
                                                                 - refuse empty keep-list
                                                                 - never touch canonical
                                                                 - dryRun => report only
                                                                 - RemovePrefix survivors
                                                                 - return per-bucket report
```

Rationale for the split: atlas-data owns object storage, so the S3 sweep belongs
there — but it must **not** reach into other namespaces. Cross-env tenant
enumeration is orchestration and stays in the CronJob; atlas-data only ever
receives an explicit keep-list. This respects the service boundary and keeps
MinIO credentials inside the one service that already holds them.

## Component 1 — atlas-data reconcile endpoint (executor)

**Route:** `POST /api/data/minio/reconcile`, operator-gated (`X-Atlas-Operator: 1`,
same guard as `tenantpurge`/`baseline`). Registered in `main.go` next to
`tenantpurge.InitResource`.

**Package:** new `services/atlas-data/atlas.com/data/minioreconcile/`
(`handler.go`, `reconcile.go`, `reconcile_test.go`). Kept separate from
`tenantpurge` because it is keep-list-driven bulk reconciliation, not a
single-tenant delete.

**Request (JSON:API `minioReconcileRequests`):**

| Field | Type | Default | Notes |
|---|---|---|---|
| `keepTenantIds` | `[]string` | — (required) | UUIDs to preserve; empty ⇒ **422 refuse** |
| `minAgeHours` | `int` | 48 | prefix eligible only if newest object older than this |
| `dryRun` | `bool` | true | when true, report only; delete nothing |

**Algorithm** (`Reconcile(ctx, l, mc, req) (Report, error)`):

1. If `len(keepTenantIds) == 0` → return `ErrEmptyKeepList` (handler → 422). An
   empty keep-list must never mean "delete everything." Mirrors the PreDelete
   hook's existing empty-list refusal.
2. Build `keep := set(keepTenantIds)`.
3. For each bucket in `{BucketWZ, BucketAssets, BucketRenders}`:
   a. Enumerate top-level tenant UUIDs under `tenants/` (list with
      `Prefix: "tenants/"`, group by the 2nd path segment).
   b. For each `uuid`:
      - Skip if `uuid ∈ keep`.
      - Skip if `uuid == canonical.TenantUUID` or
        `canonical.IsCanonical(uuid, …)` — defensive; canonical data is not
        under `tenants/` but never risk it.
      - `PrefixStats(bucket, "tenants/<uuid>/")` → count, bytes, newest
        `LastModified`.
      - If `newest > now-minAgeHours` → record `action="kept-too-new"`, continue.
      - Eligible: if `dryRun` → `action="would-delete"`; else
        `RemovePrefix(bucket, "tenants/<uuid>/")`, `action="deleted"`.
   c. Accumulate report rows.
4. Return `Report{ Buckets: [...], TotalBytes, TotalPrefixes, DryRun }`.

**Testability:** `Reconcile` depends on a narrow interface, not the concrete
`*minio.Client`:

```go
type Store interface {
    ListTenantPrefixes(ctx context.Context, bucket string) ([]string, error) // uuids under tenants/
    PrefixStats(ctx context.Context, bucket, prefix string) (minio.Stats, error)
    RemovePrefix(ctx context.Context, bucket, prefix string) error
    Cfg() minio.Config
}
```

`*minio.Client` satisfies it after adding one thin `ListTenantPrefixes` helper
(list `tenants/` with delimiter `/`, return distinct 2nd segments). Tests inject
an in-memory fake with a controllable clock (`now` injected, not `time.Now()`
directly, so the 48 h boundary is deterministic).

**Report** is returned as JSON:API `minioReconcileReports` and also logged
line-per-prefix so the CronJob's stdout captures it.

## Component 2 — `atlas-minio-reconcile` CronJob (orchestrator)

**Script:** `services/atlas-pr-bootstrap/scripts/reconcile-minio.sh`, reusing
`lib.sh` (logging, `run_phase`, tenant-enumeration idiom) and shipped in the
existing `atlas-pr-bootstrap` image — no new image or CI/bake target.

**Logic:**

1. `kubectl get ns -o name` → filter `atlas-main` + `^atlas-pr-`.
2. For each ns: `curl -fsS http://atlas-ingress.<ns>.svc.cluster.local/api/tenants`
   → `jq -r '.data[].id'`. **Fail-closed:** any non-2xx / unparseable response
   for a discovered namespace → `record_error` and **abort** (do not POST a
   partial keep-list; a missing env's tenants must never be treated as orphans).
3. Union all IDs (sort -u). Refuse to proceed if the union is empty.
4. `POST` to `http://atlas-data.atlas-main.svc.cluster.local:8080/api/data/minio/reconcile`
   with `X-Atlas-Operator: 1`, body `{keepTenantIds, minAgeHours: 48, dryRun: <flag>}`.
   `dryRun` is read from env, sourced from a ConfigMap key
   (`RECONCILE_DRY_RUN`, default `"true"`).
5. Log the returned report; non-2xx → phase fails (visible Job failure).

**K8s manifests** (`deploy/k8s/base/` + registered in overlays per
`docs/adding-a-new-service.md` discipline where applicable):

- `CronJob atlas-minio-reconcile` — `schedule: "@daily"`,
  `concurrencyPolicy: Forbid`, `restartPolicy: OnFailure`, runs the script.
- `ServiceAccount atlas-minio-reconcile` + `ClusterRole`/`ClusterRoleBinding`
  granting `list` on `namespaces` (nothing else). Reaching each env's ingress is
  plain cluster DNS — no extra RBAC.
- `ConfigMap` with `RECONCILE_DRY_RUN`. **Rollout:** ship `"true"`; after
  observing a few runs' reports, flip to `"false"` to enable real deletion.

## Component 3 — hardened PreDelete hook (B)

Edit `services/atlas-pr-bootstrap/scripts/predelete-purge.sh`:

- Wrap tenant enumeration in a bounded retry/backoff (reuse `retry` from
  `lib.sh`) so a transient atlas-tenants blip does not fail the whole purge.
- Retry each per-tenant `DELETE` a few times before marking `rc=1`.
- Emit an explicit alert-level log on final failure (so a genuinely missed
  purge is visible, not silent), preserving the existing "empty list → refuse"
  and non-zero-exit-on-failure guarantees.

With A as the true backstop, B is intentionally light — it lowers the frequency
of leaks, not the last line of defense.

## Safety properties (cross-cutting)

1. **Fail-closed keep-list** — empty (endpoint 422) or partial (orchestrator
   aborts) keep-list ⇒ no deletion.
2. **Age guard** — only prefixes whose newest object is older than 48 h.
3. **Canonical exclusion** — canonical UUID / `IsCanonical` never swept.
4. **Dry-run default** — deletion disabled until the ConfigMap flag is flipped.
5. **Observability** — every reclaim (and every would-reclaim) is reported and
   logged per-prefix with bytes.

## Testing

**atlas-data (`minioreconcile/reconcile_test.go`), in-memory fake + injected clock:**
- keep-list filtering: kept UUIDs untouched, unknown UUIDs eligible.
- 48 h boundary: object at 47 h 59 m kept-too-new; at 48 h 01 m eligible.
- empty keep-list → `ErrEmptyKeepList` (handler 422), nothing deleted.
- canonical UUID excluded even when absent from keep-list.
- `dryRun=true` deletes nothing but reports `would-delete`; `dryRun=false`
  calls `RemovePrefix`.
- multi-bucket aggregation totals.

**Orchestrator + hook (bats, existing harness):**
- union computation across mocked multi-namespace `/api/tenants`.
- fail-closed: one unreachable env ⇒ no POST, non-zero exit.
- empty union ⇒ refuse.
- PreDelete hook: retry succeeds on 2nd attempt; exhausts retries → visible fail.

**Standard gates** (per CLAUDE.md): `go test -race ./...`, `go vet ./...`,
`go build ./...` in atlas-data; `docker buildx bake atlas-data` and
`atlas-pr-bootstrap` if their `go.mod`/image inputs change; guard scripts.

## Open risks

- **xl-single vs S3 delete:** the endpoint uses the S3 API (`RemovePrefix`), the
  correct path (the manual incident used backend `rm` only because no `mc` was
  available). No backend mutation in the automated path.
- **New env mid-bootstrap:** covered by fail-closed enumeration (its ingress
  answers before tenants exist) + the 48 h age guard.
- **atlas-pr-bootstrap image scope:** it becomes a shared script toolbox rather
  than strictly PR-bootstrap. Accepted to avoid a new image/CI target; revisit
  if the image accretes more cluster-wide roles.
