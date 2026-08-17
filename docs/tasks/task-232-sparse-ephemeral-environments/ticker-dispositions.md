# Ticker dispositions (FR-6.1)

Every `time.NewTicker`-bearing file in `services/`, classified and dispositioned.
Task 41 converted the first four class-1 loops; Task 42 converts the remaining
four and dispositions the rest.

## Inventory correction: 18 -> 19

```
grep -rl "NewTicker" services | grep -v _test.go
```

returns **19** files, not the plan's 18. The 19th,
`services/atlas-configurations/atlas.com/configurations/environments/heartbeat.go`,
did not exist when design §7.2 was written — it was added by Task 19 on this
branch (environment records in `atlas-configurations`). It is dispositioned
below like every other file (§3). The plan's 18 decompose exactly as 4 (Task 41
class-1) + 4 (Task 42 class-1) + 6 (class-2) + 4 (class-3); the 19th is
genuinely additional.

---

## 1. Class 1 — converted in Task 41 (reference)

| File | Shape |
|---|---|
| `services/atlas-transports/atlas.com/transports/main.go` | serial + 1 concurrent call site (worked example) |
| `services/atlas-marriages/atlas.com/marriages/scheduler/ceremony_timeout.go` | serial |
| `services/atlas-marriages/atlas.com/marriages/scheduler/proposal_expiry.go` | serial |
| `services/atlas-asset-expiration/atlas.com/asset-expiration/task/periodic.go` | serial, session-derived (local) lister |

---

## 2. Class 1 — converted in Task 42

### `services/atlas-mts/atlas.com/mts/task/periodic.go`

**Persisted-work path, LOCAL lister.** `Sweep` discovers expired listings
cross-tenant (`database.WithoutTenantFilter`, `task/periodic.go:126`) because
the listings table stores only a `tenant_id` uuid — no region/version, so a
full `tenant.Model` can't be reconstructed for a REST-style lister. The
converted loop builds a `TenantLister` from the discovered rows themselves
(`listTenants`, deduped by `TenantId()`, region/version passed as
placeholders since `eachOwned`'s environment filter keys off `Id()` alone and
the compensating writes resolve their target tenant from the row, not the
context) and wraps the per-listing settle/expire loop in
`service.ForEachOwnedEnvironment` (serial — the pre-existing loop was a plain
`for _, lm := range expired`, no goroutines). A row whose tenant belongs to an
environment this deployment doesn't own is never visited.

`wish.DeleteExpiredWanted` (the want-ad bulk delete at the tail of the same
sweep) is **left unconverted** — it is a single bulk `DELETE ... WHERE
expires_at < ?` with no per-row tenant/environment predicate at all, already
flagged as its own, separate, pre-existing issue by
`query-scope-audit.md` ("Blocking, same shape as the frederick cleanup task").
That is an FR-8.1 tenant-scope defect, not an FR-6.1 environment-ownership gap
this task's ticker-conversion charter covers; fixing it would mean adding a
tenant/environment predicate to a query that currently has none, which is a
different, larger change than wrapping an existing per-tenant iteration.
Recorded here for visibility, not silently dropped.

`tools/scopeguard/callsite-allowlist.txt:33` was updated (line 106 -> 126) —
the conversion shifted the pre-existing `WithoutTenantFilter` call site; the
query shape is unchanged.

### `services/atlas-party-quests/atlas.com/party-quests/main.go`

**REST-backed lister** (`tenant2.NewProcessor(l, ctx).GetAll()`, same shape
as `atlas-transports`/`atlas-asset-expiration`). Both the PQ-timer ticker
(was `main.go:101`, now resolved fresh per tick via
`service.ForEachOwnedEnvironment`) and the graceful-shutdown teardown loop
(was `main.go:138`) converted; both were plain serial `for _, t := range
tenants` loops over a tenant slice loaded once at boot and closed over — the
exact C4 anti-pattern `ForEachOwnedEnvironment` exists to remove. Line numbers
had drifted from the plan's recorded `:102`/`:139`; confirmed against current
HEAD before editing.

### `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/main.go`

**Persisted-work path, LOCAL lister.** `recoverSagas` and `reapTimedOutSagas`
both read `saga.Entity` rows (`store.GetAllActive` / `store.GetTimedOut`,
themselves cross-tenant `WithoutTenantFilter` reads allowlisted at
`saga/store.go:230`/`:241` — untouched by this task, no line drift since
`main.go` was the only file edited). Unlike `atlas-mts`, each `saga.Entity`
row carries its own `TenantRegion`/`TenantMajor`/`TenantMinor`, so the shared
helper `sagaEntityTenants(entities)` reconstructs a REAL `tenant.Model` per
row (not a placeholder) for the `TenantLister`. Both loops now wrap their
per-row processing in `service.ForEachOwnedEnvironment` (serial — the
pre-existing shape was a plain `for _, e := range entities`); a row belonging
to a tenant of an environment this deployment doesn't own is never visited,
so recovery/reaping naturally partitions across baseline and override pods
without double-processing. `withSelfEnvironment` (still defined, still wired
to `saga.SagaTimers().SetEnvContext`) is unrelated to these two loops — it
backstops the saga timeout timer's fire-time environment origination
(`time.AfterFunc`-based, not `time.NewTicker`, out of this task's ticker
inventory) and was left untouched per the brief's explicit instruction not to
revert commit `685e9756b`.

### `services/atlas-login/atlas.com/login/main.go`

**LOCAL lister, built from the projection snapshot.** The republish ticker
(was `main.go:134`) previously called `publishSnapshot()` unconditionally
every second, republishing the ENTIRE projection snapshot (`state.Snapshot()`)
into the legacy `configuration` package-level cache regardless of whether
this deployment still owns every tenant in it. Converted to
`publishOwnedSnapshot`: builds a `TenantLister` from the snapshot's tenant map
(placeholder-free — `tenant.RestModel` carries real Region/MajorVersion/
MinorVersion) and filters the republished map down to tenants of environments
this deployment currently owns via `service.ForEachOwnedEnvironment`, fresh on
every tick, before calling `configuration.PublishSnapshot`.
`configuration/projection/loop.go` (the socket-listener apply loop that reads
the SAME `state`) is a separate ticker in the same service and is
**left unconverted** — see class 3 disposition below.

**Note for the next reviewer:** `atlas-character-factory`'s and
`atlas-world`'s `configuration/bridge.go` (`RunBridge`, class 3 below) do the
structurally identical thing — snapshot a projection, republish it
unconditionally on a ticker — and were NOT converted. The difference found on
inspection: `atlas-login`'s and `atlas-channel`'s tenant-config topics are the
same wire shape as `character-factory`'s/`world`'s (no environment dimension
in the envelope; `handleTenant` in both `atlas-login` and
`atlas-character-factory` applies every tenant envelope unconditionally, no
`ServiceId` gate — only the *service*-config envelope is `ServiceId`-gated).
So the underlying data these four projections mirror is environment-invariant
tenant/service *configuration*, not environment-scoped state, in all four
services alike. `atlas-login`'s ticker was still converted here because the
brief's Files section explicitly named it as one of the four class-1
conversions (and design.md's per-tenant-domain-work table row names
`atlas-login/…/main.go` specifically); the conversion is a safe narrowing
either way (it only ever *removes* tenants from what gets republished, never
adds any), so doing it does not create a correctness problem even though the
underlying data turns out not to carry an environment dimension. Flagged
here rather than silently reclassified, since the brief's explicit
file-level instruction — not my own re-derivation — is what drove the
conversion.

---

## 3. Class 2 — process-local, no change (6 files + evidence)

### `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/reservation/cache.go`

**Class 2 — process-local reservation cache. No change.**

- No tenant loop: `grep -n "tenant\." cache.go` -> no matches.
- No emitted work: `grep -n "producer\.\|Emit\|requests\." cache.go` -> no matches.
- What it does instead: `cleanupExpired` (`cache.go:106-126`) evicts entries
  from its own in-process `map[uint32]uint32`/`map[uint32]time.Time` keyed by
  item id, holding a `sync.Mutex`. No tenant or environment dimension exists
  in the cache's key space at all.

### `services/atlas-channel/atlas.com/channel/character/chakra/registry.go`

**Class 2 — process-local recovery-window registry. No change.**

- No tenant *loop*: `grep -n "tenant\." registry.go` -> lines 32/73/86/99 are
  all `tenant.Model` type references (the map key embeds a tenant), not a
  tenant-listing call.
- No emitted work: `grep -n "producer\.\|Emit\|requests\." registry.go` -> no matches.
- What it does instead: `spawnSweeper`'s ticker (`registry.go:129-144`) calls
  `r.Sweep(time.Now())` (`registry.go:111-122`), which walks the registry's
  OWN `map[Key]Entry` (Key already embeds `tenant.Model`) under its own
  mutex — no external tenant enumeration, no environment dimension; per the
  file's own comment (`registry.go:146-163`) the sweeper is deliberately a
  single process-lifetime singleton, not per-tenant/per-environment.

### `services/atlas-channel/atlas.com/channel/monster/live_mirror.go`

**Class 2 — process-local live-monster mirror. No change.**

- No tenant loop: `grep -n "tenant\." live_mirror.go` -> lines 80/98/115/133/150/160
  are all `tenant.Model`/tenant-id PARAMETERS on accessor methods, not a
  tenant-listing call.
- No emitted work: `grep -n "producer\.\|Emit\|requests\." live_mirror.go` -> no matches.
- What it does instead: `sweepLoop`'s ticker (`live_mirror.go:59-65`) calls
  `SweepStale` (`live_mirror.go:171-187`), which walks the mirror's OWN
  `perTenant map[uuid.UUID]map[uint32]LiveEntry` under its own lock —
  self-contained eviction of local process state, no external enumeration.

### `services/atlas-channel/atlas.com/channel/kafka/consumer/npc/shop/consumer.go`

**Class 2 — already single-tenant by construction. No change.**

- Tenant reference present but NOT a cross-tenant loop:
  `grep -n "tenant\." consumer.go` -> line 254,
  `t := tenant.MustFromContext(ctx)`, captured ONCE before
  `startRemoteMerchantSweep`'s ticker starts (`consumer.go:253-297`). The
  ticker body never ranges over a tenant list — it's bound to the single
  tenant its enclosing socket listener was started for. `InitHandlers` (and
  therefore this sweep) is invoked once per `(tenant, world, channel)`
  listener key, already environment/tenant-scoped by the listener bootstrap
  itself (`consumer.go:218-245`'s per-tenant dedup comment); there is no
  second, wider tenant set to filter.
- No `producer.`/cross-tenant `Emit`: `grep -n "producer\.\|Emit\|requests\." consumer.go` ->
  no matches (`session.EnableActions` at `consumer.go:292` writes to the one
  already-resolved socket session `s`, not a broadcast).

### `services/atlas-data/atlas.com/data/runtime/ingest/heartbeat.go`

**Class 2 — process-local self-heartbeat. No change.**

- No tenant loop: `grep -n "tenant\." heartbeat.go` -> no matches (atlas-data
  ingest has no tenant concept at all — it operates on WZ/version scope, not
  tenants).
- No emitted work: `grep -n "producer\.\|Emit\|requests\." heartbeat.go` -> no matches.
- What it does instead: `runHeartbeat` (`heartbeat.go:38-59`) refreshes ONE
  Redis key for `env.Self()` (`heartbeat.go:43`) — this pod's own environment
  identity, not an enumeration of environments or tenants.

### `services/atlas-data/atlas.com/data/runtime/rest/watchdog.go`

**Class 2 — process-local job-GC sweep. No change.**

- No tenant loop: `grep -n "tenant\." watchdog.go` -> no matches.
- No emitted work: `grep -n "producer\.\|Emit\|requests\." watchdog.go` -> no matches.
- What it does instead: `Watchdog.Run`'s ticker (`watchdog.go:38-49`) lists
  k8s Jobs labeled `labelIngest=true` in this pod's own namespace
  (`watchdog.go:60-62`) and reads/writes Redis keys scoped to `env.Self()`
  (`watchdog.go:98,121-122,134`) — a per-pod k8s/Redis maintenance sweep, not
  a tenant or cross-environment iteration.

---

## 4. Class 3 — control-plane, no change (4 files + evidence)

Per design §7.2 Step 5 (D5): these are environment-scoped, not
tenant-scoped. Each replays a compacted topic from `FirstOffset` under a
per-process consumer group; a sparse override projects its own
environment's configuration because its `SERVICE_ID` selects its own row
(design C1), so no per-environment iteration is needed.

### `services/atlas-channel/atlas.com/channel/configuration/projection/loop.go`

`ApplyLoop.Run`'s 250ms ticker (`loop.go:47-76`) diffs successive
`State.Snapshot()`s and drives `listener.Registry` Add/Drain. The service-half
of the projection is gated by `SERVICE_ID`: `main.go:176`
(`serviceId := uuid.MustParse(os.Getenv("SERVICE_ID"))`), wired to the
Subscriber at `main.go:186` (`ServiceId: serviceId`); a service envelope/
tombstone not matching this id is skipped (mirrors login's `handleService`,
below). No change.

### `services/atlas-login/atlas.com/login/configuration/projection/loop.go`

Same shape, same `ApplyLoop.Run` 250ms ticker (`loop.go:47-76`).
`SERVICE_ID` resolved at `main.go:54`
(`serviceId := uuid.MustParse(os.Getenv("SERVICE_ID"))`), wired to the
Subscriber at `main.go:64` (`ServiceId: serviceId`); a non-matching service
envelope/tombstone is skipped in `handleService`
(`configuration/projection/subscriber.go:99-124`). No change — this is the
socket-listener lifecycle driver, distinct from `main.go`'s own republish
ticker (converted, §2 above).

### `services/atlas-character-factory/atlas.com/character-factory/configuration/bridge.go`

`RunBridge`'s ticker (`bridge.go:19-47`) republishes `state.Snapshot()`
(tenant-config only — "this service runs no per-tenant socket listeners"
per `configuration/projection/envelope.go:1-8`) into the legacy config cache.
No `SERVICE_ID` gate exists here at all (unlike the two `loop.go` files
above) — the Subscriber (`configuration/projection/subscriber.go:20-27`) has
no `ServiceId` field, and `handleTenant` applies every tenant envelope
unconditionally. The tenant-config topic it mirrors carries no environment
dimension (`TenantEnvelope` at `configuration/projection/envelope.go:15-21`
has no environment field) — it is the tenant's canonical config, identical
regardless of which environment reads it, so there is no per-environment set
to filter against. No change.

### `services/atlas-world/atlas.com/world/configuration/bridge.go`

Identical shape and identical `RunBridge` (`bridge.go:20-48`) to
character-factory's, invoked at `main.go:127`
(`configuration.RunBridge(rt.Context(), l, state.Snapshot, time.Second, configuration.ReinitChangedRates(l))`).
Same no-`SERVICE_ID`/no-environment-dimension evidence applies. `world`'s own
additional `ReinitChangedRates` onChange hook (`bridge.go:55-61`) only
diffs/reacts to VALUE changes in the same unfiltered tenant map — it adds no
new tenant/environment enumeration. No change.

### `services/atlas-configurations/atlas.com/configurations/environments/heartbeat.go` (the 19th file — see inventory correction above)

`StartHeartbeat`'s 30s ticker (`heartbeat.go:18-33`) calls
`p.Republish(envlib.Self())` — republishing THIS pod's OWN environment record
(`envlib.Self()`, `heartbeat.go:27`), used purely as compacted-topic liveness
(design §4.3, Task 18's `StaleAfter`). No `tenant.` reference at all, no
per-environment or per-tenant loop, no cross-environment iteration — `Self()`
is definitionally single-environment. Evidenced same as the other class-3
files by direct inspection (no separate grep needed: the entire function body
is 15 lines and contains exactly one non-ticker call). No change.

---

## 5. Summary

| Class | Count | Disposition |
|---|---|---|
| 1 (converted, Task 41) | 4 | `ForEachOwnedEnvironment`/`ForEachOwnedEnvironmentConcurrently` |
| 1 (converted, Task 42) | 4 | `ForEachOwnedEnvironment` (all four serial, matching pre-existing shape) |
| 2 (process-local) | 6 | No change; evidenced above |
| 3 (control-plane) | 5 (4 planned + the 19th) | No change; evidenced above |
| **Total** | **19** | |

No class-2 or class-3 file reclassified to class 1 during this task's
evidence pass. The one file whose classification was actively reconsidered —
`atlas-login/main.go`'s republish ticker, against its structural twins
`atlas-character-factory`/`atlas-world`'s `bridge.go` — is called out in §2
rather than silently converted or silently left alone.
