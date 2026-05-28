# PR-Env Teardown Leak Fixes — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-27
---

## 1. Overview

A live cluster audit on 2026-05-27 found that atlas's ephemeral per-PR environments leak two classes of state on teardown. Both were confirmed against the running `bee` cluster, not just inferred from code.

**Leak #1 — Redis keys that skip env-namespacing.** The shared library `libs/atlas-redis` exists to prefix every key with the per-env hash: `KeyPrefix()` returns `<ATLAS_ENV>:atlas` (or just `atlas` when `ATLAS_ENV` is empty). The PostDelete cleanup (`services/atlas-pr-bootstrap/scripts/cleanup.sh` → `do_drop_redis`) reclaims a torn-down env's keys by deleting `${ATLAS_ENV}:*`. But several services build keys directly on the raw `goredis` client with bare namespaces (`coordinator:*`, `drops:all`, `reactors:all`, `transport:*`, `channel:tenants`, `invite:active-tenants`) and never call `KeyPrefix()`. Those keys (a) are never matched by `${ATLAS_ENV}:*`, so they leak into the shared `redis.home` instance on every teardown, and (b) — for the global singletons that carry neither tenant nor env in the key — are read and written by **every PR env and `atlas-main` simultaneously**, an isolation bug, not merely a leak. Live evidence: `redis.home` held 13 stale `transport:channels:<dead-tenant>:GMS:83.1` entries plus a shared `channel:tenants` set, while properly-prefixed keys (`main:atlas:*`) were clean and `drop-redis` had left zero `<hash>:atlas:*` orphans.

**Leak #2 — MinIO per-tenant data that cleanup silently fails to remove.** The PostDelete cleanup's `do_drop_tenant_storage` phase is supposed to delete each torn-down tenant's MinIO prefixes (`tenants/<uuid>/` in `atlas-wz`, `atlas-assets`, `atlas-renders`). But it runs in a lifecycle slot where it cannot reliably do its job and is allowed to fail invisibly: it reads the tenant list from the env's `atlas-data-<env>` Postgres DB and shells out to `mc`, and on any missing prerequisite (creds, endpoint, unreadable `tenant_baselines`, missing `mc`) it logs "skipping" and returns **0** — so the cleanup Job reports success while gigabytes remain. The only cluster-wide catch-all, `sweep-orphans.sh --minio`, is an operator-run CLI with **no CronJob** scheduling it. Live evidence: 5 orphaned tenant prefixes totalling ~11.6 GiB (≈11.4 GiB reclaimed manually during the audit). The root structural problem is timing: by the time the Argo CD PostDelete hook fires, the per-PR namespace — including `atlas-data` — has already been pruned, so the cleanup cannot call atlas-data's own purge endpoint and instead reimplements a fragile subset of it in bash.

This task fixes both leaks at their structural root: route **all** Redis key construction through `libs/atlas-redis`, and move tenant-storage teardown to a **PreDelete** hook that calls atlas-data's existing purge endpoint while the env is still alive, backed by a scheduled orphan-sweep CronJob for the cases where the hook cannot run.

The work spans `libs/atlas-redis`, the services `atlas-guilds`, `atlas-drops`, `atlas-reactors`, `atlas-transports`, `atlas-world`, `atlas-invites` (plus `atlas-rates` and `atlas-maps`), the `atlas-pr-bootstrap` teardown scripts, and `deploy/k8s/` overlays. It introduces no new shared services and no Postgres schema migrations. Because some changes (the PreDelete hook's ServiceAccount/RBAC, the CronJob's SA, and the `minio-root-creds`/`atlas-pr-cleanup-env` it consumes) live in the sibling `cluster-infra` repo, this task produces a coordination note for those, mirroring the pattern already used by `postdelete-cleanup.yaml`.

## 2. Goals

Primary goals:
- Eliminate the Redis key leak and the cross-environment key collision by making every Redis key any service writes begin with `libs/atlas-redis` `KeyPrefix()`.
- Prevent recurrence of the bare-key class with an automated regression guard.
- Reclaim `atlas-main`'s existing orphaned bare Redis keys once, after the fix lands.
- Eliminate the MinIO per-tenant leak by purging tenant storage via atlas-data's `DELETE /api/data/tenants/{id}` in a PreDelete hook (while atlas-data is alive), so teardown reliably removes Postgres rows and MinIO objects across all three buckets.
- Add a scheduled `sweep-orphans.sh --minio --apply` CronJob as a cluster-wide backstop for envs whose PreDelete purge could not run (atlas-data unhealthy, hook failed, manual namespace deletion).
- Make any remaining best-effort cleanup path fail loudly (record a non-zero/warning phase) instead of silently succeeding, so a future regression is observable.

Non-goals:
- The object-id silent-collision fallback bug (TODO.md / task-019) — a separate Redis correctness issue, not a teardown leak.
- The task-071 data-pipeline followups (task-076).
- Changes to MinIO bucket layout, the `tenants/<uuid>/` / `shared/` scope scheme, or the baseline/restore mechanics.
- Per-PR sweep of non-MinIO resources via the CronJob (DBs/topics/groups/Redis/ghcr) — those are already handled by `do_drop_*` in PostDelete; the CronJob scope is `--minio` only.
- Fixing the unrelated `set -e`/`pipefail` exit-code bug in `tools/task-numbers.sh` (noted during this task; spin out if desired).
- Changing how `ATLAS_ENV` is derived or how PostDelete reclaims correctly-prefixed Redis keys.

## 3. User Stories

- As an **operator**, when a PR is closed I want all of that env's Redis keys, MinIO objects, and Postgres rows gone, so shared infra (`redis.home`, MinIO, `postgres.home`) does not accumulate dead-env state indefinitely.
- As an **operator**, I want a teardown that fails to clean storage to show up as a failed/visible step, not a green success, so I can act on it.
- As an **operator**, I want a scheduled sweep that reclaims any orphaned MinIO tenant data even when a teardown hook never ran, so a single missed teardown does not become permanent leaked storage.
- As a **developer on `atlas-main`**, I want PR envs to never read or write keys in main's Redis keyspace, so a PR env cannot corrupt main's runtime caches/registries (e.g. `channel:tenants`, `transport:channels`).
- As a **developer adding Redis usage to a service**, I want a check that fails my build if I construct a key without the shared prefix, so I cannot reintroduce this leak.

## 4. Functional Requirements

### 4.1 Redis key namespacing (Leak #1)

- FR-1.1 Every Redis key written or read by any atlas service MUST be derived from `libs/atlas-redis` `KeyPrefix()` (directly, or via the shared `Registry`/`TenantRegistry`/`Index`/`namespacedKey`/`CompositeKey` helpers). `atlas-monsters` (uses `NewTenantRegistry`) is the reference implementation.
- FR-1.2 The following confirmed-bypassing call sites MUST be migrated to prefixed keys:
  - `atlas-guilds`: `coordinator/registry.go` — `coordinator:active`, `coordinator:char:<tenantKey>:<id>`, `coordinator:agreement:<uuid>`.
  - `atlas-drops`: `drop/registry.go` — `drops:all`, `drop:<tenant>:<id>`, and the map set keys.
  - `atlas-reactors`: `reactor/registry.go` — `reactors:all`, `reactor:<tenant>:<id>`.
  - `atlas-transports`: `instance/*_registry.go`, `channel/registry.go` — `transport:characters`, `transport:instances`, `transport:instance:<id>`, `transport:instance:<id>:chars`, `transport:route:<tenant>:<route>`, `transport:channels:<tenantKey>`.
  - `atlas-world`: `channel/registry.go` — `channel:tenants`.
  - `atlas-invites`: `invite/registry.go` — `invite:active-tenants`.
- FR-1.3 The broader audit ("All bypassing keys") MUST also resolve:
  - `atlas-rates`: the bare `item:<templateId>` key in `character/item_tracker.go`.
  - `atlas-maps`: the hardcoded scan literal `atlas:maps:spawn:%s:*` in `map/monster/registry.go:296` — confirm the write-side key construction and make the scan pattern consistent with `KeyPrefix()` (a read/write mismatch here is a latent correctness bug as well as a leak vector).
  - Any additional call site surfaced by the FR-1.5 guard when first run.
- FR-1.4 Cross-tenant-within-env semantics MUST be preserved. Global singleton keys (e.g. `channel:tenants`, `drops:all`, `reactors:all`, `coordinator:active`, `transport:instances`, `transport:characters`, `invite:active-tenants`) aggregate across tenants **within one env**; prefixing them with `KeyPrefix()` (→ `<env>:atlas:…`) keeps that aggregation intact while isolating per env. The fix MUST NOT introduce per-tenant scoping where the key was intentionally env-global.
- FR-1.5 A regression guard MUST fail (CI / `go test`) if a service constructs a Redis key that does not originate from `libs/atlas-redis`. The exact mechanism (e.g. a source-scanning test that flags raw `goredis` client calls with string-literal keys not routed through the shared helpers, with an explicit allowlist for the helpers themselves) is for design; it MUST run in the normal verification path the CLAUDE.md build steps describe.
- FR-1.6 A one-time, idempotent cleanup MUST remove `atlas-main`'s now-orphaned bare keys after the fix is deployed (the `transport:channels:*` dead-tenant entries, `channel:tenants`, and any other bare keys the services stop using). This is an operational runbook step, scriptable and safe to re-run.

### 4.2 MinIO tenant-storage teardown (Leak #2)

- FR-2.1 Per-tenant storage cleanup MUST move out of the PostDelete `cleanup.sh` (`do_drop_tenant_storage`) and into a **PreDelete** hook that runs while the per-PR namespace, `atlas-data`, and `atlas-ingress` are still alive.
- FR-2.2 The PreDelete hook MUST enumerate the env's tenant UUID(s) from a live source (atlas-tenants `GET /api/tenants` or atlas-data) and, for each, call `DELETE /api/data/tenants/{id}` against the in-namespace ingress. That endpoint runs `tenantpurge.Purge`, which deletes the tenant's Postgres rows and best-effort MinIO objects.
- FR-2.3 The remaining PostDelete phases (`drop-dbs`, `drop-topics`, `drop-groups`, `drop-redis`, `drop-images`, `drop-dns`, `drop-branch`) are unchanged and continue to run after prune.
- FR-2.4 If the PreDelete purge cannot enumerate tenants or a purge call fails, the hook MUST exit non-zero (or otherwise surface a visible failure), NOT report success. No silent skip.
- FR-2.5 A `CronJob` MUST run `sweep-orphans.sh --minio --apply` on a schedule as the cluster-wide backstop, in the `argocd` namespace, reusing the `atlas-pr-cleanup` ServiceAccount, the `atlas-pr-cleanup-env` ConfigMap, and the `minio-root-creds` Secret (all confirmed present in `argocd` on 2026-05-27). It MUST protect active `atlas-main` tenants (already implemented in `sweep_minio` via the main tenant-list cross-reference and the `MINIO_TENANT_SAFETY_WINDOW_SEC` age window).
- FR-2.6 If any best-effort tenant-storage path is retained anywhere (e.g. a fallback in PostDelete), a missing prerequisite MUST be recorded as a non-zero/warning phase via the existing `run_phase`/`record_error` framework, not a `return 0` skip.

## 5. API Surface

No new endpoints. The PreDelete hook consumes existing endpoints:

- `GET /api/tenants` (atlas-tenants, in-namespace) — enumerate the env's tenant UUID(s). JSON:API; `data[].id` is the UUID.
- `DELETE /api/data/tenants/{id}` (atlas-data, in-namespace via atlas-ingress) — installed by `tenantpurge.InitResource`; routes to `tenantpurge.Purge(ctx, l, db, mc, tenantID)` (deletes Postgres rows + best-effort MinIO objects). Design MUST confirm the exact request shape (tenant/operator headers, success status code, idempotency on already-purged tenant) by reading `services/atlas-data/atlas.com/data/tenantpurge/handler.go` and `purge.go`, and MUST confirm `Purge` covers all three buckets and the `tenants/<uuid>/` prefix.

## 6. Data Model

- No Postgres schema migrations.
- Redis key-shape change (not a schema): keys move from bare (`transport:channels:…`) to env-prefixed (`<env>:atlas:transport:channels:…`). On `atlas-main`, services begin writing the prefixed keys and stop referencing the bare ones; the bare keys become dead and are removed by the FR-1.6 one-time cleanup. No data needs to be copied/migrated — these are runtime caches/registries that repopulate.

## 7. Service Impact

- **`libs/atlas-redis`** — reference for the migration; may gain a small non-tenant helper if env-global keys need a cleaner constructor than raw `KeyPrefix()` concatenation. Houses the regression-guard test if implemented as a lib-level test (design decision).
- **`atlas-guilds`, `atlas-drops`, `atlas-reactors`, `atlas-transports`, `atlas-world`, `atlas-invites`** — migrate all Redis key construction to the shared helpers (FR-1.2).
- **`atlas-rates`, `atlas-maps`** — fix the stray/scan-literal keys (FR-1.3).
- **`atlas-pr-bootstrap`** — `scripts/cleanup.sh` (remove/relocate `do_drop_tenant_storage`; FR-2.6 if a fallback remains); a new PreDelete entrypoint script (or reuse of the bootstrap image with a new command); `sweep-orphans.sh` unchanged unless design finds a gap; bats tests updated.
- **`deploy/k8s/overlays/pr`** — new PreDelete hook manifest (annotated `argocd.argoproj.io/hook: PreSync`? no — `PreDelete`), wired into the `pr` kustomization. **`deploy/k8s/`** (or overlays) — new CronJob manifest for the sweep.
- **cluster-infra (sibling repo)** — coordination note: PreDelete hook ServiceAccount/RBAC (in-namespace, to reach atlas-ingress; likely no special RBAC beyond network), CronJob ServiceAccount (reuse `atlas-pr-cleanup`), and confirmation that `minio-root-creds` + `atlas-pr-cleanup-env` remain reflected into `argocd`. Mirror the `dev/cluster-infra-coordination/` pattern.

## 8. Non-Functional Requirements

- **Isolation/security (multi-tenancy):** after the fix, no PR env may read or write any key in another env's or main's Redis keyspace. This is the primary correctness property; the regression guard (FR-1.5) defends it.
- **Observability:** teardown storage-cleanup failures MUST be visible (failed hook / non-zero phase / log at `error`), per FR-2.4/FR-2.6. The CronJob's runs should be inspectable (it already logs structured JSON via `lib.sh`).
- **Idempotency:** PreDelete purge, the CronJob sweep, and the FR-1.6 cleanup MUST all be safe to re-run (already true for `sweep_minio` and `Purge`'s best-effort semantics; verify for the new hook).
- **Safety:** the CronJob MUST never delete an active `atlas-main` tenant's data — guaranteed by `sweep_minio`'s allowlist + safety window; design MUST confirm the window default (7200s) is appropriate for the cron cadence.
- **Performance:** no hot-path latency change from #1 (key construction cost is unchanged — same string building, now via a helper). PreDelete adds bounded time to teardown (one REST call per tenant, typically one tenant).
- **Build/verification:** per CLAUDE.md, every changed Go module passes `go test -race ./...`, `go vet ./...`, `go build ./...`, and `docker buildx bake atlas-<svc>` for each service whose `go.mod` is touched.

## 9. Open Questions

- **OQ-1 (PreDelete timing):** Confirm an Argo CD `PreDelete` hook Job runs *before* the resources-finalizer prunes the namespace, and that it can reach atlas-ingress/atlas-data while they are still Running. Verify Argo CD version supports `PreDelete` (the repo already uses `PostSync` and `PostDelete`).
- **OQ-2 (purge contract):** Exact `DELETE /api/data/tenants/{id}` request shape — required headers (tenant/operator), success code, behavior when the tenant has no data. Does `tenantpurge.Purge` cover `atlas-wz`, `atlas-assets`, AND `atlas-renders` under `tenants/<uuid>/`? (Read `purge.go`.)
- **OQ-3 (multi-tenant envs):** Can a PR env ever have more than one tenant? The bootstrap creates one canonical tenant, but the hook should enumerate and purge all tenants the env owns, not assume one.
- **OQ-4 (CronJob cadence & mode):** Schedule (proposed: hourly) and `--apply` vs alert-only. Hourly `--apply` with the 2h safety window means an orphan is reclaimed within ~1–3h and an in-flight bringup is never touched. Confirm acceptable.
- **OQ-5 (guard mechanism):** Source-scanning test vs. a lint/`go vet`-style analyzer for FR-1.5; where it lives and how it's invoked in CI.
- **OQ-6 (PostDelete fallback):** Keep a hardened (loud-failing) `do_drop_tenant_storage` in PostDelete as defense-in-depth, or remove it entirely and rely on PreDelete + CronJob? (Leaning remove, since it cannot work post-prune and the CronJob covers the gap.)
- **OQ-7 (cluster-infra sequencing):** Does the PreDelete hook need any RBAC beyond default namespace networking? Confirm the sibling-repo changes required and their merge ordering relative to this PR.

## 10. Acceptance Criteria

Leak #1:
- [ ] Every call site in FR-1.2 and FR-1.3 builds keys via `libs/atlas-redis`; no service constructs a Redis key from a bare string literal on the raw client.
- [ ] The FR-1.5 regression guard exists, runs in CI/`go test`, and fails on a deliberately re-introduced bare key.
- [ ] On a fresh PR env, every Redis key the env writes begins with `<ATLAS_ENV>:atlas`; after teardown, `redis.home` has zero keys for that env's `ATLAS_ENV` and zero new bare keys attributable to it.
- [ ] The FR-1.6 one-time cleanup script removes main's existing orphaned bare keys and is idempotent.
- [ ] Changed Go modules pass `go test -race ./...`, `go vet ./...`, `go build ./...`, and `docker buildx bake atlas-<svc>`.

Leak #2:
- [ ] A PreDelete hook purges each tenant's Postgres rows and MinIO objects via `DELETE /api/data/tenants/{id}` while the env is alive; on failure it exits non-zero (no silent success).
- [ ] `do_drop_tenant_storage` is removed from (or hardened to loudly fail in) the PostDelete path; no code path returns success while skipping storage cleanup.
- [ ] A `CronJob` runs `sweep-orphans.sh --minio --apply` on the agreed schedule, protects active main tenants, and reclaims a planted orphan prefix within one safety-window+cadence interval.
- [ ] After closing a PR end-to-end on a test env, MinIO contains no `tenants/<uuid>/` prefix for that env's tenant in any of the three buckets, and the env's `atlas-data-<env>` rows are gone.
- [ ] `atlas-pr-bootstrap` bats tests updated for the relocated/removed phase and any new hook script; the image builds.
- [ ] A `dev/cluster-infra-coordination/` note documents the required sibling-repo changes (PreDelete SA/RBAC, CronJob SA, secret/configmap reflection) with merge ordering.

---

### Audit evidence (for context; not part of scope)

- Redis (`redis.home`, DB 0): 13 stale `transport:channels:<dead-tenant>:GMS:83.1` + 1 `channel:tenants`; `main:atlas:*` clean; zero leftover `<hash>:atlas:*`.
- MinIO: 5 orphan tenant prefixes (~11.6 GiB) in `atlas-wz`/`atlas-assets`; ~11.4 GiB reclaimed manually during the audit; only the live main tenant (`ec876921…`) now remains.
- Prerequisites confirmed present in `argocd` ns on 2026-05-27: `minio-root-creds` (reflected 12:30 UTC), `atlas-pr-cleanup-env` (with `MINIO_ENDPOINT`). No orphaned `atlas-pr-*` Applications/namespaces.
- atlas-data purge endpoint confirmed: `DELETE /data/tenants/{id}` → `tenantpurge.Purge(ctx, l, db, mc, tenantID)` (`services/atlas-data/atlas.com/data/tenantpurge/`).
