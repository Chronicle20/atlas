# Redis key-format cutover (Tasks 4-7)

Tasks 4-7 moved eleven Redis namespaces across four services (`atlas-storage`,
`atlas-monsters`, `atlas-summons`, `atlas-npc-shops`, `atlas-doors`) from
`libs/atlas-redis`'s bare `Registry`/`KeyedSet` API onto its tenant-scoped
`TenantRegistry`/`TenantKeyedSet` equivalents, per Decision D7 (data-plane
Redis state must be scoped by the tenant-key segment because sparse
environments share `main`'s keyspace — the `ATLAS_ENV` prefix that isolated
Redis per-environment goes away under D1). Every migrated namespace's rendered
key changed shape. **The keys a populated Redis holds before this deploys are
not the keys the migrated code reads after** — they are orphaned, not
migrated, and nothing rewrites them. No build, no test, and no
`tools/verify.sh` run can see this: every module compiles and every suite
passes against a fresh (empty) Redis. The effect appears only at deploy time,
against a Redis that already has data in it.

## Verified table

| Namespace | Service | Format changed | State class | Orphaning impact |
|---|---|---|---|---|
| `door`, `door-field`, `door-owner`, `door-town` | atlas-doors | yes: `atlas:door:<tenantId>:<id>` → `atlas:door:<tenantId>:<region>:<major>.<minor>:<id>` (all four namespaces, same shape) | live state, `Put` (no TTL), no DB (`atlas-doors` has no Postgres persistence) | player must recast an in-play door |
| `monster`, `monster-map` | atlas-monsters | yes: `atlas:monster:<tenantId>:<id>` → `atlas:monster:<tenantId>:<region>:<major>.<minor>:<id>` | live world state, `reg.Put`/`mapIdx.Add` (no TTL), no DB (`atlas-monsters` has no Postgres persistence) | spawned mobs untracked until the spawn system's next cycle |
| `monster-cooldown`, `monster-attack-cooldown` | atlas-monsters | yes, same shape | ephemeral, `reg.PutWithTTL` | none — TTL-bounded, re-derived on next use |
| `monster-puppet`, `monster-puppet-field` | atlas-monsters | yes, same shape | live skill state, `reg.Put` (no TTL) | one puppet loses server-side tracking until its own timer expires |
| `summon`, `summon-map`, `summon-owner` | atlas-summons | yes, same shape (all three already embedded the tenant by hand; the tenant *segment's shape* still changed — region/version was added) | live world state, `reg.Put` (no TTL) | active summons untracked until natural expiry |
| `npc-shop:consumables` | atlas-npc-shops | yes: `atlas:npc-shop:consumables:<tenantId>` → `atlas:npc-shop:consumables:<tenantId>:<region>:<major>.<minor>:rechargeable` | ephemeral, cache-aside (`GetConsumables` re-fetches and re-`Put`s on a cache miss) | none — repopulates on next read |
| `npc-context` | atlas-storage | yes — gained a tenant segment it never had: `atlas:npc-context:<characterId>` → `atlas:npc-context:<tenantId>:<region>:<major>.<minor>:<characterId>` | ephemeral, `reg.PutWithTTL` | none — rewritten on next interaction |

Every row above was confirmed against the migration commit's diff
(`05dd24392` atlas-storage, `0c23f3e64` atlas-monsters, `982ed082d`
atlas-summons/atlas-npc-shops, `bffd89034` atlas-doors) and the key-builder in
`libs/atlas-redis/keys.go` (`namespacedKey`, `tenantEntityKey`, `TenantKey`).
No row required correction against the reviewer's original table.

## Task 9 additions

Task 9 closed the last three bare-constructor call sites the analyzer would
otherwise have had to allowlist (`rediskeyguard-analyzer.md` step 1-2
decisions).

| Namespace | Service | Format changed | State class | Orphaning impact |
|---|---|---|---|---|
| `drop-timer` | atlas-monsters | yes: hand-built `atlas:drop-timer:<tenantId>:<uniqueId>` (bare `Registry[string,...]`) → `atlas:drop-timer:<tenantId>:<region>:<major>.<minor>:<uniqueId>` (`TenantRegistry[uint32,...]`) | live per-monster drop-rate-limit state, `reg.Put` (no TTL), no DB | one monster's next-drop timer resets; re-armed on its next hit/spawn cycle, per `drop_timer_registry.go`'s own doc comment |
| `hidden-character` | atlas-monsters | yes: hand-built `atlas:hidden-character:<tenantId>:<characterId>` (bare `Registry[string,...]`) → `atlas:hidden-character:<tenantId>:<region>:<major>.<minor>:<characterId>` (`TenantRegistry[uint32,...]`) | live GM-hide flag, `reg.Put` (no TTL), no DB | a GM-hidden character reverts to visible until re-hidden; self-corrects on the reconciliation sweep (`GetAllAcrossTenants`) or the next `Add` call |
| `maps:spawn` | atlas-maps | yes: `atlas:maps:spawn:<tenantId>:<world>:<channel>:<map>:<instance>` (bare `KeyedHash`, tenant UUID hand-embedded in the key fn) → `atlas:maps:spawn:<tenantId>:<region>:<major>.<minor>:<world>:<channel>:<map>:<instance>` (`TenantKeyedHash`) | live spawn-point cooldown state, `initializeScript`/`reserveEligibleScript` Lua (no TTL), no DB — cooldowns are derived from static template data via `dp.GetSpawnableSpawnPoints`, not persisted | `InitializeForMap`'s `EXISTS`-gated Lua script re-seeds the map's spawn points fresh (`NextSpawnAt` = now) on the first read after deploy — points become immediately spawnable rather than honoring their pre-deploy cooldown, self-corrects within one spawn cycle |

Confirmed against `drop_timer_registry.go` (`NewTenantRegistry`, `r.Register`),
`character/hidden/registry.go` (`NewTenantRegistry`, `r.Add`), and
`map/monster/registry.go` (`NewTenantKeyedHash`, `InitializeForMap`).

## Task 9C additions

Task 9C closed the two `atlas-transports` DATA-PLANE gaps the `rediskeyguard`
analyzer found once its Task 9B worktree-detection fix started actually
running: `character_registry.go`'s env-global `Hash` (a live cross-tenant
collision — tenant A's character 12345 and tenant B's character 12345 wrote
the same hash field) and `instance_registry.go`'s three env-global
`Set`/`Registry`/`KeyedHash` fields, which made every sweep
(`GetExpiredBoarding`/`GetExpiredTransit`/`GetAllActive`/`GetStuck`)
cross-tenant even though `instanceId` is a fresh random UUID and never
collided. `atlas-transports` has no Postgres persistence anywhere in the
service (verified: no `gorm`/`database/sql` import in the module) — every row
below is Redis-only state.

| Namespace | Service | Format changed | State class | Orphaning impact |
|---|---|---|---|---|
| `transport:characters` | atlas-transports | yes: `atlas:transport:characters` (one env-global `Hash`, field = bare `characterId`) → `atlas:transport:characters:<tenantId>:<region>:<major>.<minor>` (one `TenantHash` per tenant, field still bare `characterId`) | live state, `chars.Set` (no TTL), no DB | a character mid-boarding/mid-transit at deploy time loses its `IsInTransport`/`GetInstanceForCharacter` lookup; the old hash's field can no longer be reached under the new key. The stranding self-heals on their next `HandleMapEnter`/`HandleLogout` (the character registry no longer reports them in transport, so those handlers treat them as not-in-transport and no-op) — worst case the player keeps whatever transit-map buff was applied until it naturally expires; there is no re-entrant morph or duplicate boarding, since `StartTransport`'s double-transport guard (`cr.IsInTransport`) also reads the same (now-empty) hash and allows a fresh `StartTransport` |
| `transport:instances` | atlas-transports | yes: `atlas:transport:instances` (one env-global `Set` of instance ids, `all`) → `atlas:transport:instances:<tenantId>:<region>:<major>.<minor>` (one `TenantSet` per tenant) | live state, `all.Add`/`all.Remove` (no TTL), no DB | the four sweep methods (`GetExpiredBoarding`, `GetExpiredTransit`, `GetAllActive`, `GetStuck`) stop finding pre-deploy instance ids in the new per-tenant Set, so an instance already in flight at deploy time is never ticked to completion/cancellation by `TickBoardingExpiration`/`TickArrival`/`TickStuckTimeout`/`GracefulShutdown` — its boarding characters are stuck (see `transport:characters` row above for their self-heal path) until an operator manually clears them, since nothing re-adds a pre-existing instance id to the new Set |
| `transport:instance` | atlas-transports | yes: `atlas:transport:instance:<instanceId>` (env-global `Registry`, keyed only by instance id) → `atlas:transport:instance:<tenantId>:<region>:<major>.<minor>:<instanceId>` (`TenantRegistry`) | live per-instance metadata (route, state, timestamps), `meta.Put` (no TTL), no DB | same instance is now unreachable under the old key; `loadMetadata`/`GetInstance` return not-found for it post-deploy. Same self-heal path as `transport:instances` above — the instance's own characters fall out of transport tracking rather than being warped anywhere incorrect |
| `transport:instance:chars` | atlas-transports | yes: `atlas:transport:instance:chars:<instanceId>` (env-global `KeyedHash`, one hash per instance) → `atlas:transport:instance:chars:<tenantId>:<region>:<major>.<minor>:<instanceId>` (`TenantKeyedHash`) | live per-instance character roster, `chars.Set`/`chars.Del` (no TTL), no DB | the character roster for an in-flight instance becomes unreachable under the old key; `AddCharacter`/`RemoveCharacter`/`loadCharacters` for that instance id see an empty roster post-deploy. Combined with the `transport:instance` row, an in-flight instance effectively vanishes rather than corrupting into another tenant's — there is no cross-tenant leak at any point in the cutover, only a one-time loss of in-flight-transport tracking that clears itself as each affected character's own session events (map enter/exit, logout) run |

`transport:route` (`routes`, the per-route `TenantKeyedSet[uuid.UUID]`) was
already tenant-scoped before Task 9C and is unchanged by this task.

Confirmed against `character_registry.go` (`NewTenantHash`, `chars.Set`/`Get`/
`Del`/`Exists`) and `instance_registry.go` (`NewTenantSet`, `NewTenantRegistry`,
`NewTenantKeyedHash`, `storeMetadata`/`loadMetadata`/`filterInstances`).

## Task 9D additions

Task 9D closed the two `atlas-guilds` DATA-PLANE gaps `rediskeyguard` found
in `coordinator/registry.go`: an env-global `Set` (`active`, holding pending
guild-creation agreement-id strings) and an env-global `Registry`
(`agreements`, agreement-id → `Model`). Not colliding today only because
`agreementId` is a fresh `uuid.New()` — luck, not design; `charAgree`
(`coordinator:char`, characterId → agreement-id) was already a
`TenantRegistry` before this task and is unchanged. `atlas-guilds` *does*
have Postgres (the `guild`/`member`/`title` tables), but the `coordinator`
package that owns these two namespaces imports no `gorm`/database dependency
at all (verified: no `gorm`/`.db` reference anywhere under
`coordinator/*.go`) — the in-flight guild-creation agreement is pure Redis
state, distinct from the guild itself, which is Postgres-backed once
created. Both rows below are Redis-only.

| Namespace | Service | Format changed | State class | Orphaning impact |
|---|---|---|---|---|
| `coordinator:active` | atlas-guilds | **retired, not migrated**: the `active *atlas.Set`/`*atlas.TenantSet` field, its constructor, and both call sites (`Initiate`'s `active.Add`, `Respond`'s disagree-branch `active.Remove`) were deleted outright — there is no new key shape because nothing writes or reads this namespace anymore under any shape | was live state, `active.Add`/`active.Remove` (no TTL), no DB; now dead | this row is a **retirement**, not a migration, so the standard "orphaned until repopulated, self-heals on next write" story does not apply. `active` was purely a write-path index: `GetExpired`/`GetExpiredAcrossTenants` never read it (confirmed: `grep -rn "\.Members(" services/atlas-guilds/... --include="*.go"` returns no hits anywhere in the service), so removing it changes no runtime behavior — `GetExpiredAcrossTenants` already reads `agreements` directly (see next row). The pre-deploy env-global key `atlas:coordinator:active` (a single `Set` of agreement-id strings, bounded by however many agreements were pending at the moment this deploy lands) becomes permanently orphaned garbage: nothing will ever read, rewrite, or grow it again, under either the old or a new key shape, because the code path that touched it no longer exists. Operationally harmless — it is a single small SET (member count bounded by in-flight agreements at deploy time, which is small and does not grow further) that nothing references; a one-off `DEL atlas:coordinator:active` is optional cleanup, not required for correctness |
| `coordinator:agreement` | atlas-guilds | yes: `atlas:coordinator:agreement:<agreementId>` (env-global `Registry[uuid.UUID,Model]`) → `atlas:coordinator:agreement:<tenantId>:<region>:<major>.<minor>:<agreementId>` (`TenantRegistry[uuid.UUID,Model]`) | live in-flight guild-creation agreement (members, responses, age), `agreements.Put` (no TTL), no DB | an agreement in progress at deploy time becomes unreachable under the old key: `Respond` (a member accepting/declining) fails with "agreement not found" (`registry.go:82-85`), and the expiration ticker's `GetExpiredAcrossTenants` sweep (reading `agreements` directly via `GetAllAcrossTenants`) never sees it either since it too scans only the new key shape — it can never expire the stranded agreement to unblock its members. This does **not** self-heal: `charAgree` (`coordinator:char`, already tenant-scoped and unaffected by this cutover) keeps pointing each affected member at the now-orphaned agreement id, so `Initiate`'s "already attempting guild creation" guard (`registry.go:39-43`) keeps rejecting a fresh guild-creation attempt from that member, while `Respond` can never clear the `charAgree` entry because it errors out at the `agreements.Get` not-found check (`registry.go:82-85`) before it ever reaches the code that resets `charAgree` to `uuid.Nil` (`registry.go:93-97`). The affected members (at most the ones mid-agreement at the moment of deploy) are stuck unable to initiate or join a new guild-creation attempt until an operator manually deletes their stale `coordinator:char:<tenantId>:<region>:<major>.<minor>:<characterId>` key — no corruption or cross-tenant leak, just a manual-cleanup dependency, and the blast radius is bounded to whoever had a pending agreement at the exact deploy instant |

Confirmed against `coordinator/registry.go` (`NewTenantRegistry`,
`Initiate`/`Respond`/`GetExpiredAcrossTenants`, and the deleted `active`
field/constructor/call sites) and `guild/task.go` (the expiration ticker,
`context.Background()`, no tenant in context).

## What to expect on the deploy that lands Tasks 4-7

A brief, one-time world-state reset limited to the "live state, no TTL, no DB"
rows above: active doors, spawned monsters, monster puppets, and summons. On
first read after the deploy, the code looks up the new key shape, finds
nothing, and treats it as absent — same as any cold cache. Normal gameplay
(the spawn system's cycle, a door being recast, a puppet's or summon's own
expiry timer) recreates all of it within one cycle. **No player data is at
risk.** Nothing in this table is a system of record: there is no Postgres
table behind any of these namespaces (`atlas-doors`, `atlas-monsters`, and
`atlas-summons` carry no Postgres persistence at all), and the two ephemeral
rows (`npc-context`, `npc-shop:consumables`, and the two cooldown namespaces)
were always designed to be rebuilt from a miss.

## What NOT to do

Do not write a key-migration script to carry old keys forward under the new
shape. The old keys are stale pointers to entities that normal gameplay
already knows how to recreate; copying them forward would resurrect world
state that should have expired or been recast, which is worse than the brief
reset above. This was a deliberate call, not an oversight — do not add one as
a follow-up unless the reset behavior above is observed to *not* self-heal
(see below).

## How to tell this apart from a real incident

The signature of the expected cutover effect:

- Misses cluster in the first read after the deploy lands — a burst of
  "monster not found" / door-not-found / cache-miss-and-refetch behavior
  concentrated in the minutes immediately following rollout, not a steady
  ongoing rate.
- Each miss self-clears within one spawn/expiry cycle: a missing door gets
  recast by the next player action, a missing monster reappears on the next
  spawn-system pass, a missing summon/puppet is gone until its owner's next
  cast, a missing `npc-context`/`npc-shop:consumables` entry is filled by the
  next read that hits it.
- **No error-level logs from the miss itself.** `libs/atlas-redis`'s
  `Get`/`entityKey` path returns a distinct `ErrNotFound` sentinel for a
  missing key (`tenant_registry.go:44-53`, `registry.go:39-50`) — a plain
  "not there," not a wrapped client/connection error. Callers treat it as an
  ordinary, expected outcome rather than a failure: e.g.
  `monster.ClearAggro` (`services/atlas-monsters/atlas.com/monsters/monster/processor.go:1786-1789`)
  matches `errMonsterNotFound` explicitly and logs it at `Debug`
  ("monster no longer exists; dropping") rather than `Error`, per that
  function's own FR-4.6 comment ("a command naming a monster that no longer
  exists is logged and dropped, not retried into an error loop"). The
  `npc-context` and `npc-shop:consumables` caches don't log a miss at all —
  they just fall through to repopulate.
- **Genuine Redis failures return a different error and get logged at
  `Error`.** A connection/timeout/command failure from `go-redis` does not
  satisfy `errors.Is(err, ErrNotFound)` — it comes back as the wrapped
  `fmt.Errorf("redis get: %w", err)` path instead (`tenant_registry.go:50-53`).
  Call sites elsewhere in the same packages that handle a real Redis-call
  error do log it at `Error`/`Warn` level (e.g.
  `atlas-doors`'s `expiry_task.go:57` — `t.l.WithError(err).Errorf("door
  expiry sweep failed")` — and `atlas-monsters`'s `status_task.go:44,95`,
  `picker_task.go:88`), and unlike the cutover reset those errors keep
  recurring on every tick until the underlying connectivity problem is
  fixed — they do not self-clear.

**The practical check:** if the misses stop within one spawn/expiry cycle of
the deploy and grep for `Error` in the affected service's logs around that
window comes back empty (or only shows the ordinary Debug/Warn "no longer
exists, dropping" lines), it's the expected cutover reset. If misses are
still happening past that window, or if `WithError(err).Error(...)`/`Errorf`
lines are showing up from the Redis call sites, treat it as a real incident —
check Redis connectivity, not the key format.
