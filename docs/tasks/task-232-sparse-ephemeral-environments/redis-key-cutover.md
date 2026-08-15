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
