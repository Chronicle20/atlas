# PR-Env Teardown Leak Fixes — Design

Version: v1
Status: Proposed
Created: 2026-05-27
PRD: `docs/tasks/task-045-pr-teardown-leak-fixes/prd.md`
---

## 1. Summary

Two structural leaks bleed dead-env state into shared infra on every PR teardown. This design fixes both at the root:

- **Leak #1 (Redis):** Route *every* Redis key through `libs/atlas-redis` so all keys carry the per-env `KeyPrefix()`. Because almost all leaking keys are Redis **SETs and HASHes** accessed via raw `goredis` calls — which the lib's existing JSON-KV `Registry`/`TenantRegistry` cannot model — we **extend `libs/atlas-redis` with typed set/hash registries** and migrate every bypassing call site onto them. A **`go vet`-style analyzer** then bans keyed operations on the raw `*goredis.Client` outside `libs/atlas-redis`, making regressions a build failure.
- **Leak #2 (MinIO):** Move per-tenant storage teardown out of the post-prune `cleanup.sh` and into an **Argo CD `PreDelete` hook** that purges each tenant via atlas-data's `DELETE /api/data/tenants/{id}` while the env is still alive. The fragile `do_drop_tenant_storage` PostDelete phase is **deleted**. A **6-hourly `sweep-orphans.sh --minio --apply` CronJob** is the cluster-wide backstop, with its safety allowlist **extended to protect live PR-env tenants**, not just `atlas-main`.

No new shared services, no Postgres migrations, no Kafka changes. Singleton cluster infra (PreDelete SA needs, sweep CronJob + RBAC) is coordinated with the sibling `cluster-infra` repo via a `dev/cluster-infra-coordination/` note.

## 2. Decisions (locked during design)

| # | Decision | Chosen | Rejected alternatives |
|---|----------|--------|-----------------------|
| D1 | How to route SET/HASH/global keys through the lib | **Extend `libs/atlas-redis` with typed Set/Hash registries**; migrate all sites; ban raw client | Small key-builder helper only; inline `CompositeKey(KeyPrefix(),…)` |
| D2 | FR-1.5 regression-guard mechanism | **`go vet`-style `analysis.Analyzer`** (`tools/rediskeyguard`) | Per-module source-scanning test; in-lib source scanner |
| D3 | Fate of PostDelete `do_drop_tenant_storage` | **Remove entirely** | Keep, hardened to fail loudly |
| D4 | Sweep CronJob cadence & mode | **`0 */6 * * *`, `--apply`** | Hourly `--apply`; hourly alert-only-first |
| D5 | Sweep safety for live-but-idle PR envs | **Allowlist live PR-env tenants** (enumerate `atlas-pr-*` namespaces, union their `/api/tenants`) | Rely on 2h window only; widen window to days |

D1/D2 chose the more rigorous end deliberately: typed registries let the analyzer enforce a single decidable rule ("no keyed op on the raw client outside the lib") instead of trying to trace key-string provenance.

---

## 3. Leak #1 — Redis key namespacing

### 3.1 The shape problem

`libs/atlas-redis` already prefixes correctly for the patterns it covers (`keys.go:27` `KeyPrefix()` → `atlas` or `<env>:atlas`; `Registry`, `TenantRegistry`, `Index`, `TTLRegistry`, `IDGenerator`, `Lock`). The reference implementation is atlas-monsters (`monster/information/cache.go:140` `NewTenantRegistry`).

But the leaking sites are **not** JSON-KV gets/puts. They are:

- **Env-global SETs** — `drops:all`, `reactors:all`, `channel:tenants`, `invite:active-tenants`, `transport:instances`, `coordinator:active` (raw `SAdd`/`SRem`/`SMembers`).
- **Env-global HASH** — `transport:characters` (raw `HSet`/`HDel`/`HExists`/`HGet`).
- **Keyed families of SETs/HASHes** — `coordinator:agreement:<uuid>`, `drops:map:<t>:<field>`, `reactors:map:<t>:<field>`, `reactor:spot:*` / `reactor:cd:*` (per-map, scan-by-prefix), `transport:instance:<id>:chars`, `transport:route:<t>:<route>`, `transport:channels:<tenantKey>`, `atlas:maps:spawn:<t>:<field>`.
- **Per-entity marshaled values** — `drop:<t>:<id>`, `reactor:<t>:<id>`, `coordinator:char:<tk>:<id>`, `transport:instance:<id>` (these *do* fit the existing `Registry`/`TenantRegistry`).

The lib has no env-global set, env-global hash, or keyed-set/keyed-hash type. Per D1 we add them, parallel in style to the existing generics (constructor takes `client`, `namespace`, and where needed a `keyFn func(K) string`; all keys flow through the same `namespacedKey`/`KeyPrefix()` machinery in `keys.go`).

### 3.2 New `libs/atlas-redis` API

All key formats below are shown post-prefix; the literal prefix is `KeyPrefix()` = `atlas` (main, `ATLAS_ENV` empty) or `<env>:atlas` (PR env).

**Env-global (no tenant):**

| Type | Constructor | Key format | Backing | Ops |
|------|-------------|-----------|---------|-----|
| `Set` | `NewSet(client, namespace)` | `<prefix>:<namespace>` | Redis SET | `Add`,`Remove`,`Members`,`IsMember`,`Size` |
| `Hash` | `NewHash(client, namespace)` | `<prefix>:<namespace>` | Redis HASH | `Set`,`Get`,`Del`,`Exists`,`GetAll` |
| `KeyedSet[K]` | `NewKeyedSet[K](client, namespace, keyFn)` | `<prefix>:<namespace>:<keyFn(k)>` | Redis SET | per-key `Add`/`Remove`/`Members`/… |
| `KeyedHash[K]` | `NewKeyedHash[K](client, namespace, keyFn)` | `<prefix>:<namespace>:<keyFn(k)>` | Redis HASH | per-key hash ops |

**Tenant-scoped:**

| Type | Constructor | Key format | Backing | Ops |
|------|-------------|-----------|---------|-----|
| `TenantSet` | `NewTenantSet(client, namespace)` | `<prefix>:<namespace>:<tenantKey>` | Redis SET | tenant-scoped set ops |
| `TenantKeyedSet[K]` | `NewTenantKeyedSet[K](client, namespace, keyFn)` | `<prefix>:<namespace>:<tenantKey>:<keyFn(k)>` | Redis SET | per-key set ops + `ScanKeys(t)`/`Clear(t)` |
| `TenantKeyedHash[K]` | `NewTenantKeyedHash[K](client, namespace, keyFn)` | `<prefix>:<namespace>:<tenantKey>:<keyFn(k)>` | Redis HASH | per-key hash ops + `ScanKeys(t)`/`Clear(t)` |

- `tenantKey` is the existing `TenantKey(t)` (`keys.go:31`) → `<uuid>:<region>:<major>.<minor>`.
- Members are `string` by default; provide `Uint32` convenience wrappers mirroring the existing `Uint32Index` (`index.go`) where a service stores numeric members (drops/reactors map sets).
- `TenantKeyedSet`/`TenantKeyedHash` expose a **tenant-scoped sub-prefix SCAN** (`ScanKeys(ctx, t)` → all `<keyFn>` segments for that tenant) so the per-map / per-tenant enumerations currently done with raw `client.Scan` move *inside* the lib. This is what lets the analyzer (3.4) ban raw `Scan` in services.
- Existing `Registry[K,V]` / `TenantRegistry[K,V]` are reused unchanged for the marshaled-value cases.

These types are thin wrappers over the same `keys.go` helpers; no change to `KeyPrefix()` or `TenantKey()`.

### 3.3 Call-site migration map

| Service | File | Current key (bare) | New type | New key (main / PR-env) |
|---|---|---|---|---|
| atlas-guilds | `coordinator/registry.go` | `coordinator:active` | `Set` ns=`coordinator:active` | `atlas:coordinator:active` / `<env>:atlas:coordinator:active` |
| atlas-guilds | `coordinator/registry.go` | `coordinator:agreement:<uuid>` | `KeyedSet[uuid]` ns=`coordinator:agreement` | `…:coordinator:agreement:<uuid>` |
| atlas-guilds | `coordinator/registry.go` | `coordinator:char:<tk>:<id>` | `TenantRegistry[uint32,Model]` ns=`coordinator:char` | `…:coordinator:char:<tk>:<id>` |
| atlas-drops | `drop/registry.go` | `drops:all` | `Set` ns=`drops:all` | `…:drops:all` |
| atlas-drops | `drop/registry.go` | `drop:<t>:<id>` | `TenantRegistry[uint32,Model]` ns=`drop` | `…:drop:<tk>:<id>` |
| atlas-drops | `drop/registry.go` | `drops:map:<t>:<field>` | `TenantKeyedSet[field]`(uint32 members) ns=`drops:map` | `…:drops:map:<tk>:<field>` |
| atlas-reactors | `reactor/registry.go` | `reactors:all` | `Set` ns=`reactors:all` | `…:reactors:all` |
| atlas-reactors | `reactor/registry.go` | `reactor:<t>:<id>` | `TenantRegistry[uint32,Model]` ns=`reactor` | `…:reactor:<tk>:<id>` |
| atlas-reactors | `reactor/registry.go` | `reactors:map:<t>:<field>` | `TenantKeyedSet[field]` ns=`reactors:map` | `…:reactors:map:<tk>:<field>` |
| atlas-reactors | `reactor/registry.go` | `reactor:cd:<t>:<field>:…` (scan) | `TenantKeyedHash[field]` ns=`reactor:cd` (field-x-y → ts) | `…:reactor:cd:<tk>:<field>` |
| atlas-reactors | `reactor/registry.go` | `reactor:spot:<t>:<field>:…` (scan-by-prefix) | `TenantKeyedHash[field]` ns=`reactor:spot` | `…:reactor:spot:<tk>:<field>` |
| atlas-world | `channel/registry.go` | `channel:tenants` | `Set` ns=`channel:tenants` | `…:channel:tenants` |
| atlas-invites | `invite/registry.go` | `invite:active-tenants` | `Set` ns=`invite:active-tenants` | `…:invite:active-tenants` |
| atlas-transports | `instance/instance_registry.go` | `transport:instances` | `Set` ns=`transport:instances` | `…:transport:instances` |
| atlas-transports | `instance/instance_registry.go` | `transport:instance:<id>` | `Registry[uuid,Model]` ns=`transport:instance` | `…:transport:instance:<id>` |
| atlas-transports | `instance/instance_registry.go` | `transport:instance:<id>:chars` | `KeyedHash[uuid]` ns=`transport:instance:chars` | `…:transport:instance:chars:<id>` |
| atlas-transports | `instance/instance_registry.go` | `transport:route:<t>:<route>` | `TenantKeyedSet[uuid]` ns=`transport:route` | `…:transport:route:<tk>:<route>` |
| atlas-transports | `instance/character_registry.go` | `transport:characters` | `Hash` ns=`transport:characters` | `…:transport:characters` |
| atlas-transports | `channel/registry.go` | `transport:channels:<tk>` | `TenantSet` ns=`transport:channels` | `…:transport:channels:<tk>` |
| atlas-rates | `character/item_tracker.go` | manual `KeyPrefix()+…:*` scan | `TenantKeyedSet`/`Index` per character (lib owns scan) | `…:<ns>:<tk>:<characterId>:*` via lib |
| atlas-maps | `map/monster/registry.go` | write `<KeyPrefix()>:maps:spawn:…`; **scan literal `atlas:maps:spawn:%s:*` (L296)** | `TenantKeyedHash[field]` ns=`maps:spawn` (lib owns the scan) | `…:maps:spawn:<t>:<field>` |

Notes:
- **atlas-maps is the latent-correctness fix:** write side already uses `KeyPrefix()` but the scan literal (`registry.go:296`) hardcodes `atlas:`, so a PR env's write (`<env>:atlas:maps:spawn:…`) is never matched by the scan — a read/write mismatch, not just a leak. Routing both through `TenantKeyedHash` (lib-owned scan) makes them consistent by construction. Keep the existing `mapKey.Tenant` UUID-based composition if it must match an existing on-disk shape, or move to `TenantKey(t)`; either is acceptable since spawn data repopulates.
- **Key-shape change is intentional and safe.** Several per-entity keys move from a raw `tenant.Id().String()` segment to the full `TenantKey(t)` (uuid:region:ver). These are runtime caches/registries that repopulate from the source of truth or events; no data migration. The pre-fix bare keys become dead and are reclaimed by the FR-1.6 one-time cleanup (3.5).
- **Member-format consistency:** where a global index set (`drops:all`, `reactors:all`) stores composite members used to reconstruct entity keys, the migration must keep the member encoding aligned with the new per-entity key (impl task; covered by registry round-trip tests).

### 3.4 FR-1.5 regression guard — `go vet`-style analyzer (D2)

**New tool module:** `tools/rediskeyguard/` with its own `go.mod` (module `github.com/Chronicle20/atlas/tools/rediskeyguard`):

- `analyzer.go` — a `*golang.org/x/tools/go/analysis.Analyzer`.
- `cmd/rediskeyguard/main.go` — `singlechecker.Main(Analyzer)` (also usable as a `-vettool`).
- `analyzer_test.go` + `testdata/` — `analysistest`-driven unit tests (one "bad" package that must be flagged, one "good" package that must not).

**Rule (decidable, AST-only):** report any call to a method on a receiver of type `*github.com/redis/go-redis/v9.Client` (and `redis.Pipeliner`) where the method is in the keyed-command set — `Set,Get,Del,Exists,Expire,Scan,Keys,SAdd,SRem,SMembers,SIsMember,SCard,HSet,HGet,HDel,HExists,HGetAll,HKeys,…` — **unless** the enclosing package import path is `github.com/Chronicle20/atlas/libs/atlas-redis` (the sole allowlist). Passing the `*Client` as a value (e.g. `InitRegistry(client)`) is fine; only *keyed method calls* on it are flagged. This converts the leak into "no service may call a keyed command on the raw client; use an atlas-redis type."

**Why this rule over provenance-tracing:** with D1's typed registries, every legitimate keyed op already lives in the lib, so the ban is total and unambiguous. No allowlist of "good" call patterns to maintain, no string-literal heuristics.

**Invocation / CI wiring (OQ-5):**
- The tool is **standalone** — its own `go.mod`, deliberately **not** added to the root `go.work` and **not** `COPY`'d into the shared `Dockerfile`. (Adding it to `go.work` without a Dockerfile `COPY` would break every service image build — exactly the `go.work`/Dockerfile coupling CLAUDE.md warns about. Services do not import it, so it has no place in their build graph.)
- A new script `tools/redis-key-guard.sh` builds the binary once (`cd tools/rediskeyguard && go build -o …`) then runs it over each Go service module (`rediskeyguard ./...` invoked with that module's directory as the working dir, so `go/packages` resolves the target module's `go.work` context). Non-empty diagnostics → non-zero exit.
- Added to the verification path: a new CI step plus a line in `CLAUDE.md`'s build/verification list ("5. `tools/redis-key-guard.sh` clean"). It runs alongside `go vet ./...`.

### 3.5 FR-1.6 — one-time reclamation of main's orphaned bare keys

After the fix deploys, `atlas-main` (which runs with `ATLAS_ENV` empty → prefix `atlas`) will write the correctly-prefixed keys (`atlas:drops:all`, …) and stop touching the bare ones (`drops:all`, `channel:tenants`, `transport:channels:*`, …). Those bare keys become dead.

- **New operator script** `services/atlas-pr-bootstrap/scripts/reclaim-main-bare-keys.sh`, mirroring `sweep-orphans.sh` ergonomics: list-only by default, `--apply` to delete; logs via `lib.sh`.
- It deletes **only an explicit allowlist of bare namespaces** the services stop using — `channel:tenants`, `drops:all`, `reactors:all`, `coordinator:active`, `coordinator:agreement:*`, `coordinator:char:*`, `invite:active-tenants`, `transport:instances`, `transport:characters`, `transport:instance:*`, `transport:route:*`, `transport:channels:*`, `drop:*`, `reactor:*`, `reactors:map:*`, `drops:map:*`, `reactor:cd:*`, `reactor:spot:*`. It must **never** match `atlas:*` or `<hash>:atlas:*` (those are live prefixed keys). `atlas-maps` is excluded — its write side already used `KeyPrefix()`, so on main it wrote the correct `atlas:maps:spawn:*` and has no bare orphans.
- Idempotent (`DEL` of absent keys is a no-op), safe to re-run, targets `redis.home` main DB 0. Documented as a one-shot runbook step gated behind the deploy.

---

## 4. Leak #2 — MinIO tenant-storage teardown

### 4.1 PreDelete purge hook (FR-2.1, FR-2.2, FR-2.4)

**Timing (OQ-1):** Argo CD runs `PreDelete` hooks during Application deletion, **before** the resources-finalizer prunes the namespace. The repo already uses `PreSync`/`PostSync`/`PostDelete` (`overlays/pr/*`, `overlays/pr-cleanup/postdelete-cleanup.yaml`), so the controller version supports lifecycle hooks; `PreDelete` is the same 2.x-era feature as `PostDelete`. The hook Job is created **in the `atlas-pr-<N>` namespace**, so atlas-ingress / atlas-data / atlas-tenants are still Running and reachable in-namespace. (Plan step: confirm the live cluster's Argo CD version honors `PreDelete`.)

**New script** `services/atlas-pr-bootstrap/scripts/predelete-purge.sh` (added to the `Dockerfile` `COPY` block alongside the other scripts; uses `lib.sh` logging + `run_phase`/`record_error`/`summarize_phases`):

1. `GET ${INGRESS}/api/tenants` (`Accept: application/vnd.api+json`) → `jq -r '.data[].id'`. Confirmed routing: ingress `location ~ ^/api/tenants` → `atlas-tenants.<ns>.svc:8080`. Enumerate **all** ids (OQ-3 — bootstrap creates one canonical tenant, but the hook purges every tenant the env owns). Fetch failure or empty list → `record_error` → non-zero exit (no silent skip; an env always has ≥1 tenant).
2. For each id: `DELETE ${INGRESS}/api/data/tenants/<id>` with header `X-Atlas-Operator: 1`. Confirmed routing: ingress `location ~ ^/api/data` → `atlas-data.<ns>.svc:8080` (preserves `$request_uri`), reaching atlas-data's `/api/data/tenants/{id}` (`main.go:66` prefix `/api/` + `tenantpurge/handler.go:23` route). Contract (OQ-2, from `tenantpurge/handler.go`+`purge.go`): requires `X-Atlas-Operator: 1` (else 403); success **202 Accepted**; deletes 7 Postgres tables in a txn + best-effort removes `tenants/<uuid>/` from `atlas-wz`,`atlas-assets`,`atlas-renders`; refuses the canonical tenant with 403 (N/A — PR envs use non-canonical tenants); **idempotent** (DELETE is a no-op on already-purged rows, `RemovePrefix` is idempotent → re-run returns 202).
3. Any non-2xx (other than the canonical 403 which would itself be an error for a PR env) → `record_error`; `summarize_phases` returns non-zero so the hook Job fails visibly.

**Manifest** `deploy/k8s/overlays/pr/predelete-purge.yaml`, wired into `overlays/pr/kustomization.yaml`:
- `argocd.argoproj.io/hook: PreDelete`, `argocd.argoproj.io/hook-delete-policy: HookSucceeded` (failed jobs are *retained* for inspection — the visible-failure requirement).
- `namespace: atlas-pr-PLACEHOLDER_PR_NUMBER` (the per-PR namespace, like the rest of `overlays/pr`), so in-namespace ingress DNS resolves.
- Image `atlas-pr-bootstrap` (CI bumps per-PR), `command: ["/atlas/predelete-purge.sh"]`, `backoffLimit: 0`, `restartPolicy: Never`.
- Env: `PR_NUMBER` (logging/env-hash) and `ATLAS_INGRESS_BASE` = `http://atlas-ingress.atlas-pr-PLACEHOLDER_PR_NUMBER.svc.cluster.local` (matches the host `sync-bootstrap.yaml` already uses).
- **ServiceAccount:** default namespace SA. The hook needs only in-cluster networking (no Kubernetes API), so no special RBAC (OQ-7). Documented in the coordination note for confirmation.

**Failure semantics:** a failing `PreDelete` hook blocks Application deletion and surfaces the error in Argo CD — the desired visibility. If atlas-data is genuinely unhealthy, deletion stalls until an operator force-deletes; the CronJob backstop (4.3/4.4) then reclaims the storage. This is the intended FR-2.5 "hook failed / manual namespace deletion" path.

### 4.2 Remove `do_drop_tenant_storage` from PostDelete (D3, FR-2.3, FR-2.6)

`do_drop_tenant_storage` cannot work post-prune — it reads `tenant_baselines` from `atlas-data-<env>`, which `do_drop_dbs` (or the namespace prune) has already removed (`cleanup.sh:99-113`), and every missing-prerequisite branch returns `0` (silent success). With purge moved to PreDelete and the CronJob as backstop, it is **deleted outright**:

- Remove `do_drop_tenant_storage` (`cleanup.sh:81-141`) and its `PHASES` entries (`cleanup.sh:332-341`).
- The "drop-tenant-storage before drop-dbs" ordering constraint (`cleanup.sh:328-331`) goes away; remaining phases (`drop-dbs`,`drop-topics`,`drop-groups`,`drop-redis`,`drop-images`,`drop-dns`,`drop-branch`) are unchanged and still run.
- FR-2.6 is satisfied vacuously: no best-effort tenant-storage path remains in PostDelete to silently succeed. (`record_error`/`run_phase` framework in `lib.sh` is otherwise untouched.)

### 4.3 Extend `sweep_minio` allowlist to live PR-env tenants (D5, FR-2.5)

Today `sweep_minio` (`sweep-orphans.sh:318-442`) protects only `atlas-main` tenants (`ATLAS_MAIN_TENANTS_URL`) plus the `MINIO_TENANT_SAFETY_WINDOW_SEC` window (default **7200s**). A **live but idle** PR env — whose tenant UUID is not in main's list and whose WZ/asset data is static for >2h — would match the orphan criteria and get reclaimed out from under a running env. Closing this:

- After building the main allowlist, **enumerate live PR-env tenants**: `kubectl get ns -l atlas.pr-number -o jsonpath=…` (the `overlays/pr` kustomization stamps `atlas.pr-number`/`atlas.env` common labels) → for each namespace, `curl http://atlas-tenants.<ns>.svc.cluster.local:8080/api/tenants | jq -r '.data[].id'`. Union those UUIDs into the protected allowlist.
- A prefix is an orphan **only if** it belongs to no live PR namespace **and** no main tenant **and** is older than the safety window.
- **Fail-closed:** if namespace enumeration or any reachable namespace's tenant fetch fails, abort the sweep (mirrors the existing "abort if main fetch fails" at `sweep-orphans.sh:357-377`) rather than proceeding with a partial allowlist that could delete live data.
- New optional env `ATLAS_PR_NS_SELECTOR` (default `atlas.pr-number`). Requires the CronJob SA to list namespaces and reach cross-namespace `atlas-tenants` (RBAC + network — coordination note, OQ-7).

### 4.4 Sweep CronJob (D4, FR-2.5)

- Runs `sweep-orphans.sh --minio --apply` on `schedule: "0 */6 * * *"` (every 6h). With the 2h window + the 4.3 live-env allowlist, an orphan is reclaimed within ~6–8h while in-flight bringups (touched <2h ago) and all live envs (in the allowlist) are protected (OQ-4).
- Runs in the **`argocd`** namespace, reusing the `atlas-pr-cleanup` ServiceAccount, the `atlas-pr-cleanup-env` ConfigMap (`MINIO_ENDPOINT`, …), and the `minio-root-creds` Secret — all confirmed present in `argocd` on 2026-05-27 and consumed identically by `postdelete-cleanup.yaml`.
- `concurrencyPolicy: Forbid`, `successfulJobsHistoryLimit: 3`, `failedJobsHistoryLimit: 3`, image `atlas-pr-bootstrap`.
- **Ownership:** the CronJob is a **cluster-wide singleton**, not per-PR. The existing `overlays/pr-cleanup` is rendered once per PR by CI (its `postdelete-cleanup.yaml` is per-PR), so placing the singleton there would create N copies. Mirroring how its long-lived dependencies (SA, ConfigMap, Secret) are already cluster-infra-owned, the **CronJob manifest is owned by `cluster-infra`**, and this repo ships it as a coordination example (`dev/cluster-infra-coordination/sweep-orphans-cronjob.example.yaml`). The *script logic* (`sweep-orphans.sh` 4.3 changes) lives here in the `atlas-pr-bootstrap` image the CronJob consumes. This cleanly splits per-repo responsibility and resolves OQ-7 sequencing.

### 4.5 cluster-infra coordination note (OQ-7)

New `dev/cluster-infra-coordination/task-045-teardown.md` (mirroring the existing `atlas-pr-cleanup-env.example.yaml` pattern), documenting required sibling-repo changes and merge ordering:
1. **PreDelete hook:** no RBAC change — runs in-namespace with the default SA, network-only. (Confirm.)
2. **Sweep CronJob SA:** `atlas-pr-cleanup` gains RBAC to **list namespaces** cluster-wide and network egress to cross-namespace `atlas-tenants` services (for 4.3).
3. **CronJob manifest:** owned by cluster-infra (example shipped here), schedule `0 */6 * * *`, `--minio --apply`, reusing `atlas-pr-cleanup` SA + `atlas-pr-cleanup-env` + `minio-root-creds`.
4. **Confirm** `minio-root-creds` and `atlas-pr-cleanup-env` remain reflected into `argocd`.
- **Merge ordering:** this PR (image/script/PreDelete manifest) can land first; the CronJob simply won't have the new allowlist logic until the image bump propagates, and the RBAC can land in cluster-infra independently. The PreDelete hook is self-contained in this repo and needs nothing from cluster-infra to function.

---

## 5. Open questions — resolutions

| OQ | Resolution |
|---|---|
| OQ-1 PreDelete timing | Supported (same era as `PostDelete`, already in use). Hook Job runs in the per-PR namespace before prune; reaches in-namespace ingress. Plan verifies live Argo version. |
| OQ-2 purge contract | `DELETE /api/data/tenants/{id}`, header `X-Atlas-Operator: 1`, success **202**, idempotent, covers all 3 buckets under `tenants/<uuid>/` + 7 Postgres tables, refuses canonical (N/A for PR). |
| OQ-3 multi-tenant envs | Hook enumerates and purges **all** ids from `/api/tenants`; no single-tenant assumption. |
| OQ-4 cadence & mode | `0 */6 * * *`, `--apply`; 2h window + live-env allowlist protect bringups and live envs. |
| OQ-5 guard | `go vet`-style `analysis.Analyzer` in standalone `tools/rediskeyguard`; bans keyed ops on raw client outside the lib; run via `tools/redis-key-guard.sh` in CI; `analysistest`-tested. |
| OQ-6 PostDelete fallback | **Removed entirely.** |
| OQ-7 cluster-infra sequencing | PreDelete: no special RBAC. Sweep CronJob: cluster-infra-owned, needs list-namespaces RBAC + cross-ns network; reuses existing SA/ConfigMap/Secret. This PR lands independently. |

---

## 6. Testing strategy

- **`libs/atlas-redis` (miniredis unit tests, existing style):** for each new type (`Set`,`Hash`,`KeyedSet`,`KeyedHash`,`TenantSet`,`TenantKeyedSet`,`TenantKeyedHash`) — round-trip ops + **key-format assertions** that the literal key includes `KeyPrefix()` for both empty and set `ATLAS_ENV` (drive via the same `computeKeyPrefix` seam used by `keys_test.go`). Tenant-keyed types: cross-tenant isolation + `ScanKeys`/`Clear` correctness.
- **Each migrated service:** update existing registry tests to the new key formats; behavior-level tests (add/remove/lookup/scan) still pass. atlas-maps: assert write key and scan pattern are now produced by the same helper (regression for the L296 mismatch).
- **`tools/rediskeyguard`:** `analysistest` with a `testdata/bad` package (raw `client.SAdd("x", …)` → expect diagnostic) and `testdata/good` package (atlas-redis type usage → no diagnostic); confirm the lib's own package is allowlisted.
- **atlas-pr-bootstrap bats:**
  - Remove `do_drop_tenant_storage` tests from `cleanup_test.bats`.
  - New `predelete_test.bats`: mock `curl` — tenants list → two ids → two `DELETE` 202 (success exit 0); fetch failure → non-zero; a `DELETE` 500 → non-zero; idempotent re-run → 202.
  - Extend `sweep_test.bats`: mock `kubectl get ns` + per-ns `curl` so a live PR-env tenant is **protected** (not deleted) while a true orphan is; namespace-enumeration failure → sweep aborts.
  - New `reclaim_test.bats`: mock `redis-cli` — only allowlisted bare keys `DEL`'d, `atlas:*`/`<hash>:atlas:*` untouched, idempotent.
- **End-to-end (acceptance, manual on a test env):** fresh PR env → every written Redis key begins with `<env>:atlas`; close PR → `redis.home` has zero keys for that `ATLAS_ENV` and no new bare keys; MinIO has no `tenants/<uuid>/` for the env's tenant in any bucket; `atlas-data-<env>` rows gone.

## 7. Build & verification (per CLAUDE.md)

Changed Go modules: `libs/atlas-redis`, `atlas-guilds`, `atlas-drops`, `atlas-reactors`, `atlas-transports`, `atlas-world`, `atlas-invites`, `atlas-rates`, `atlas-maps`, and the standalone `tools/rediskeyguard`.

For each changed module: `go test -race ./...`, `go vet ./...`, `go build ./...`. For each **service** whose `go.mod` is touched (the 8 services + atlas-maps): `docker buildx bake atlas-<svc>` from the worktree root. `libs/atlas-redis` has no bake target (validated via its consumers). `tools/rediskeyguard` has no bake target and is **not** added to `go.work`/`Dockerfile`. Additionally run `tools/redis-key-guard.sh` (the new guard) clean across all service modules, and the `atlas-pr-bootstrap` bats suite + its image build.

No new shared lib is added to the Dockerfile/`go.work` (the new types extend the existing `libs/atlas-redis`, already wired). No Postgres migrations.

## 8. Risks & tradeoffs

- **Large lib surface + 9-service migration (from D1).** Chosen over a minimal key-builder because it enables the strict, decidable analyzer ban and removes raw-client SCAN from services. Mitigation: types are thin and parallel to existing ones; per-type unit tests; the analyzer catches anything missed.
- **Key-shape changes for per-entity keys.** Runtime caches repopulate; no migration. During a **rolling deploy**, old pods (bare keys) and new pods (prefixed keys) briefly diverge on these caches; acceptable because they rebuild from the source of truth / events (drops, reactors, registries). The bare residue is cleaned once by FR-1.6.
- **PreDelete failure blocks deletion.** Desired visibility, but a genuinely-down atlas-data stalls teardown until force-delete; the CronJob + (untouched) PostDelete phases reclaim the remainder. Net safer than today's silent success.
- **Sweep gains kubectl + cross-ns reach.** New RBAC/network surface in cluster-infra; fail-closed enumeration prevents a partial allowlist from deleting live data.
- **Analyzer is best-effort AST.** It can miss a client smuggled through an interface or reflection. The real fix is the typed migration; the analyzer is the guard rail, not the guarantee.

## 9. Out of scope (per PRD §2)

object-id silent-collision fallback (task-019); task-076 pipeline followups; MinIO bucket-layout / baseline-restore changes; per-PR non-MinIO sweep via CronJob; `tools/task-numbers.sh` exit-code bug; changes to `ATLAS_ENV` derivation or PostDelete's correctly-prefixed Redis reclamation.
