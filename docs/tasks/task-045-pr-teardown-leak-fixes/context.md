# task-045 — Context

Companion to `plan.md`. Key files, decisions, and gotchas an executor needs.

## What this task does (two independent leaks)

- **Leak #1 (Redis):** Several services build Redis keys directly on the raw
  `*goredis.Client` with bare namespaces (`drops:all`, `channel:tenants`,
  `transport:channels:*`, …). Those keys skip `libs/atlas-redis`'s env prefix
  (`KeyPrefix()` → `atlas` or `<env>:atlas`), so they (a) leak into shared
  `redis.home` on PR teardown and (b) for env-global singletons, collide across
  every PR env + `atlas-main`. Fix: extend `libs/atlas-redis` with typed
  set/hash registries, migrate every bypassing call site onto them, and add a
  `go vet`-style analyzer that bans keyed ops on the raw client outside the lib.
- **Leak #2 (MinIO):** PostDelete `cleanup.sh::do_drop_tenant_storage` runs
  after the per-PR namespace (incl. atlas-data) is already pruned, reads a
  now-gone DB, and silently `return 0`s on every missing prerequisite. Fix: move
  tenant-storage purge to an Argo CD **PreDelete** hook that calls atlas-data's
  `DELETE /api/data/tenants/{id}` while the env is alive; delete the PostDelete
  phase; add a live-PR-env-aware sweep CronJob (cluster-infra-owned) as backstop.

## Source of truth read during planning

### libs/atlas-redis (the foundation)
- `keys.go` — `KeyPrefix()` (`keys.go:27`), `TenantKey(t)` (`keys.go:31` →
  `<uuid>:<region>:<major>.<minor>`), `namespacedKey(ns, parts...)`
  (`keys.go:38` → `<prefix>:<ns>:<parts…>`), `tenantEntityKey` (`keys.go:45`),
  `tenantScanPattern` (`keys.go:49`). All new types route through these.
- `registry.go` — `Registry[K,V]` (env-global JSON-KV; reused as-is).
- `tenant_registry.go` — `TenantRegistry[K,V]` (tenant-scoped JSON-KV; `Clear`
  at `:189` is the SCAN+pipelined-DEL pattern the new `Clear` methods mirror).
- `index.go` — `Index` / `Uint32Index` (SET-based; the `Add/Remove/SMembers`
  idiom the new `Set`/`TenantKeyedSet` mirror).
- Test harness (`registry_test.go:16` `setupTestRedis(t)` → miniredis;
  `:23` `makeTenant`; `:30` `testTenant`; `tenant_registry_test.go:14`
  `newTestTenant`). New `*_test.go` files share this package and reuse these.
- `go.mod`: module `github.com/Chronicle20/atlas/libs/atlas-redis`, go 1.25.5,
  deps already include `miniredis/v2 v2.38.0`, `go-redis/v9 v9.19.0`,
  `google/uuid`. **No new deps needed for the lib.**

### Bypassing call sites (Leak #1) — verified against code, NOT the design table
The design's §3.3 table has two errors corrected here:
- **atlas-guilds** `coordinator/registry.go`:
  - `coordinator:active` = env-global SET of agreement-id strings → **`Set`**.
  - `coordinator:agreement:<uuid>` = a **marshaled `Model`** (`Set`/`Get` of
    JSON bytes), NOT a set → **`Registry[uuid.UUID, Model]`** (env-global; the
    lib type already exists). *(Design said `KeyedSet[uuid]` — wrong.)*
  - `coordinator:char:<tk>:<id>` = a **uuid string** (the agreement id), NOT a
    Model → **`TenantRegistry[uint32, string]`**. *(Design said
    `TenantRegistry[uint32, Model]` — wrong value type.)*
- **atlas-drops** `drop/registry.go`: `drops:all`→`Set`; `drop:<t>:<id>`→
  `TenantRegistry[uint32, dropEntry]`; `drops:map:…`→`TenantKeyedSet[field]`.
  Note member encoding of `drops:all` is `"<tenantUUID>:<id>"`; reconstruction
  in `GetAllDrops` builds a region-less tenant — preserved.
- **atlas-reactors** `reactor/registry.go`: `reactors:all`→`Set`;
  `reactor:<t>:<id>`→`TenantRegistry[uint32, Model]`;
  `reactors:map:…`→`TenantKeyedSet[field]`; `reactor:cd:…` and `reactor:spot:…`
  →`TenantKeyedHash[field]` (**semantic change, see below**).
- **atlas-world** `channel/registry.go`: `channel:tenants`→`Set` (members are
  JSON-marshaled `tenant.Model`).
- **atlas-invites** `invite/registry.go`: `invite:active-tenants`→`Set` (members
  are JSON-marshaled `tenant.Model`).
- **atlas-transports**: `instance/instance_registry.go` —
  `transport:instances`→`Set`, `transport:instance:<id>`→`Registry[uuid,…]`,
  `transport:instance:<id>:chars`→`KeyedHash[uuid]`,
  `transport:route:<t>:<route>`→`TenantKeyedSet[uuid]`.
  `instance/character_registry.go` — `transport:characters`→`Hash`.
  `channel/registry.go` — `transport:channels:<tk>`→`TenantSet`.
- **atlas-rates** `character/item_tracker.go` — the manual raw `client.Scan`
  in `GetAllTrackedItems` (`:216`) → re-model as `TenantKeyedHash[uint32]`
  keyed by characterId, hash field = templateId.
- **atlas-maps** `map/monster/registry.go` — write side already uses
  `KeyPrefix()` (`spawnHashKey` `:62`); the bug is the **scan literal
  `atlas:maps:spawn:%s:*` at `:296`** (`FlushTenant`) and `Reset` (`:263`).
  Uses Lua scripts (`initializeScript`, `eligibleScript`,
  `updateCooldownsScript`, `resetCooldownScript`) for atomicity — **scripts
  stay**; only raw `HGetAll`/`HSet`/`Scan`/`Del` move into the lib. See below.

### Leak #2 source
- `services/atlas-pr-bootstrap/scripts/cleanup.sh` —
  `do_drop_tenant_storage` (`:81-141`), `PHASES` (`:332-341`), ordering note
  (`:328-331`). `lib.sh` — `run_phase`/`record_error`/`summarize_phases`,
  `compute_atlas_env`, `log`. `sweep-orphans.sh` — `sweep_minio` (`:318-442`),
  MinIO-mode dispatch (`:444-453`).
- `services/atlas-data/atlas.com/data/tenantpurge/handler.go` — route
  `DELETE /data/tenants/{id}` (`:23`), requires `X-Atlas-Operator: 1` (`:35`,
  else 403), success **202** (`:53`). `purge.go` — `Purge` deletes 7 tables in
  a txn (`PurgeTables` `:20`) + best-effort `RemovePrefix` on `BucketWZ`,
  `BucketAssets`, `BucketRenders` under `tenants/<uuid>/` (`:45-53`); refuses
  canonical (`ErrCanonicalRefused`). Idempotent.
- `services/atlas-pr-bootstrap/Dockerfile` — COPY block (`:56-60`), chmod
  (`:62`); `mc`, `kubectl`, `jq`, `curl`, `redis` already installed.
- `deploy/k8s/overlays/pr/kustomization.yaml` — `resources:` (`:27-33`).
- `deploy/k8s/overlays/pr/sync-bootstrap.yaml` — the in-namespace Job pattern
  (`ATLAS_UI_BASE=http://atlas-ingress.atlas-pr-PLACEHOLDER_PR_NUMBER.svc...`,
  `:100`); the SA `atlas-pr-bootstrap` + its Role.
- `deploy/k8s/overlays/pr-cleanup/postdelete-cleanup.yaml` — the
  `argocd`-namespace hook Job (`:21-91`); consumes `atlas-pr-cleanup` SA,
  `atlas-pr-cleanup-env` ConfigMap, `minio-root-creds` Secret. The
  `minio-root-creds` envFrom (`:72-83`) becomes dead once
  `do_drop_tenant_storage` is removed.
- `dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml` — the
  coordination-note pattern to mirror.
- bats: `test/cleanup_test.bats` (`do_drop_tenant_storage` tests at `:356-447`,
  phase-count at `:356`), `test/sweep_test.bats`, `test/dockerfile_test.bats`.

### Build coupling (CLAUDE.md)
- `go.work` lists every module; the repo-root `Dockerfile` has paired COPY
  lines per lib. `docker-bake.hcl` targets are driven by
  `.github/config/services.json`.
- **`tools/rediskeyguard` is deliberately NOT added to `go.work` and NOT
  COPY'd into the root Dockerfile** — no service imports it; adding it to
  `go.work` without a Dockerfile COPY would break every image build. It has its
  own `go.mod` and is run as a standalone binary.

## Locked decisions (from design §2, plus planning corrections)

| # | Decision |
|---|----------|
| D1 | Extend `libs/atlas-redis` with typed Set/Hash registries; migrate all sites; ban raw client. |
| D2 | FR-1.5 guard = `go vet`-style `analysis.Analyzer` in standalone `tools/rediskeyguard`. |
| D3 | Delete PostDelete `do_drop_tenant_storage` entirely. |
| D4 | Sweep CronJob `0 */6 * * *`, `--apply`. |
| D5 | Sweep allowlist extended to live PR-env tenants (fail-closed). |
| **P1** | New lib types needed: `Set`, `TenantSet`, `Hash`, `KeyedHash[K]`, `TenantKeyedSet[K]`, `TenantKeyedHash[K]`. **`KeyedSet` is NOT built (YAGNI — no env-global per-key SET call site exists once `coordinator:agreement` is correctly a `Registry`).** |
| **P2** | **atlas-reactors cooldown semantics change.** Today cooldowns are individual keys with native Redis TTL; spots use `SetNX`. Moving both into per-map `TenantKeyedHash` (field = `class:x:y`) **loses per-field TTL**. New cooldown semantics: store expiry-unix-ms in the field; `IsOnCooldown` = `now < expiry`; `RecordCooldown` = `Set field`; `ClearAllCooldownsForMap` = `DeleteKey`. Spots: `TryClaimSpot` = `SetNX`, `ReleaseSpot` = `Del field`, `ClearAllSpotsForMap` = `DeleteKey`. This eliminates the raw `client.Scan` the analyzer would otherwise flag. Acceptable: reactors are a repopulating runtime cache. |
| **P3** | **atlas-maps uses the env-global `KeyedHash[character.MapKey]`** (keyFn embeds the tenant UUID + field segments), NOT a tenant-scoped type, so the `KeyPrefix()` env-prefix is preserved and the §3.5 reclaim exclusion holds for the prefix. **Correction to the design:** `character.MapKey.Tenant` is a `tenant.Model`, and the current write key uses `mk.Tenant.String()` — which is the **verbose debug form** `"Id [uuid] Region [..] Version [..]"` (`libs/atlas-tenant/tenant.go:82`), with spaces/brackets. Meanwhile `FlushTenant` scans the bare literal `atlas:maps:spawn:<bare-uuid>:*`. **These never match → `FlushTenant` deletes nothing today** (a worse bug than the `atlas:` literal the design flagged). The fix uses the **bare uuid** `mk.Tenant.Id().String()` on both write and flush, via the lib. This orphans the old verbose `atlas:maps:spawn:Id [..]` keys on main — they are atlas-prefixed (so the bare-key reclaim script must NOT touch them) and are cleaned by a one-time manual `redis-cli` command documented in Task 14's orphan note. Lua scripts run via `script.Run(ctx, client, []string{kh.Key(mapKey)}, …)` — `Run` passes the client as a value (analyzer-allowed); only keyed *method calls* on the client are banned. `FlushTenant`/`Reset` use `KeyedHash.Clear(ctx, segments...)`. |

## Gotchas

- **`go-redis` `SAdd`/`SRem` take `...interface{}`.** New lib `Set` types accept
  `...string` and convert.
- **`HGet`/`Get` return `goredis.Nil`** when absent — map to `ErrNotFound`.
- **Key-shape churn on main is intentional.** Per-entity keys that move from a
  raw `tenant.Id().String()` segment to `TenantKey(t)` orphan their old bare
  forms on main; the FR-1.6 reclaim script (Task 21) cleans them once. atlas-maps
  is excluded (P3).
- **The analyzer's banned set must include the methods actually used:** `Set Get
  Del Exists Expire Scan Keys SAdd SRem SMembers SIsMember SCard HSet HSetNX HGet
  HDel HExists HGetAll HKeys HLen SetNX`. It must **not** flag `Run`, `Eval`,
  `Pipeline`, `Watch`, or passing the client as an argument.
- **`InitRegistry(client)` signatures don't change** for any service — the
  constructors keep taking `*goredis.Client` and build the lib types internally,
  so no consumer/`main.go` wiring changes. (Verify each `Init*` caller still
  compiles.)
- **bats phase-count assertion** in `cleanup_test.bats:356` ("8 phases") drops to
  7 when `drop-tenant-storage` is removed.
- **`tools/task-numbers.sh` exit-code bug** is explicitly out of scope (PRD §2).

## Verification matrix (per CLAUDE.md)
Changed Go modules: `libs/atlas-redis`, `atlas-guilds`, `atlas-drops`,
`atlas-reactors`, `atlas-transports`, `atlas-world`, `atlas-invites`,
`atlas-rates`, `atlas-maps`, and standalone `tools/rediskeyguard`.
- Per module: `go test -race ./...`, `go vet ./...`, `go build ./...`.
- Per **service** whose `go.mod` was touched (8 services + atlas-maps = the 9
  with bake targets): `docker buildx bake atlas-<svc>` from the worktree root.
- `tools/redis-key-guard.sh` clean across all service modules.
- `bats services/atlas-pr-bootstrap/test/` green; bootstrap image builds.
- `libs/atlas-redis` has no bake target (validated via consumers).
