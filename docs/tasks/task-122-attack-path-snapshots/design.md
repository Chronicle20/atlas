# task-122 — Attack-Path Snapshots (PS-1): Design

Status: v1, for review
Inputs: `prd.md` (approved), `event-coverage.md` (FR-2 audit, committed alongside this document)

---

## 1. Summary of the chosen design

Five coordinated pieces, all inside atlas-channel except one bounded producer change:

1. **Session-scoped character snapshot** (`character/snapshot/` package): a tenant-scoped singleton registry keyed by character id holding three independently-validated components — **core** (the undecorated `character.Model`), **skills** (`[]skill.Model`), **inventory** (`inventory.Model`) — plus a locally-fed **position** overlay. The read API composes them into the same decorated `character.Model` that `GetById(InventoryDecorator, SkillModelDecorator)` returns today, so every downstream consumer (planner, damage pipeline, broadcast writers) is untouched.
2. **Kafka maintenance**: additive handlers on the existing character/skill/asset/compartment/buff consumers that update or invalidate snapshot components. Rich events apply in place (absolute values only — idempotent under redelivery); thin events invalidate the affected component; the next read refetches *only that component* over REST and backfills under a generation check.
3. **One FR-2.4(a) producer change**: atlas-character populates the existing-but-mostly-nil `Values` map on **every** `STAT_CHANGED` emission. Without it, each MP-consuming swing self-invalidates the snapshot it just used (attack → CHANGE_MP → STAT_CHANGED(Mp, nil) → invalidate), reducing the win to "1 GET per swing" instead of zero.
4. **In-process TTL cache for atlas-data skills** (`data/skill/cache.go`): task-060 semantics (5m positive / 30s negative / ErrNotFound-only negative caching / env kill-switch), in-process per task-120's design. Serves the attack `se`, the MP-Eater effect, and the writer's per-recipient mastery lookups.
5. **Monster reads via the task-120 mirror, extended with X/Y**, and restructured so one memoized monster resolve per damage entry serves both reflect and MP Eater (FR-4.2).

Buffs join the snapshot (fully event-covered, rich payloads). Effective stats stay on REST with the existing per-swing memoization (no event exists — see event-coverage.md §6). Position is fed from atlas-channel's **own movement fold**, which is strictly fresher than today's REST projection of the same data (event-coverage.md §2).

## 2. What the attack path reads (FR-1 read-set inventory)

Verified field-level inventory; paths relative to `services/atlas-channel/atlas.com/channel/`. Entry point `processAttack` (`socket/handler/character_attack_common.go:266`); melee/ranged/magic/touch handlers all funnel here (touch maps to `AttackTypeEnergy`, `character_attack_touch.go:17`).

### 2.1 `character.Model` — one fetch per swing = **3 HTTP GETs**

`cp.GetById(cp.InventoryDecorator, cp.SkillModelDecorator)` (`character_attack_common.go:271-272`) = base `GET {CHARACTERS}/characters/{id}` (`character/requests.go:20-22`) + `GET {INVENTORY}/characters/{id}/inventory` (`character/processor.go:72-78`; single call — compartments/assets arrive as JSON:API includes, `inventory/rest.go:34-97`) + `GET {SKILLS}/characters/{id}/skills` (`character/processor.go:149-155`).

| Field | Read at | Frequency |
|---|---|---|
| `Id()` | projectile planner `character_attack_projectile.go:62,73,78,85,92,102,108,131,140`; all four writer bodies | per swing + per broadcast recipient |
| `Level()` | writer bodies (melee/ranged `:21`, magic/energy `:19`) | per broadcast recipient |
| `JobId()` | `character_attack_common.go:215` (MP Eater); writer `preComputeAttackValues` `socket/writer/character_attack_common.go:41` | per damaged monster (magic) / per recipient |
| `Skills()` | `character_attack_common.go:221,282`; writer `:23,41` (mastery via `getMasteryFromSkill` `:194-207`) | per swing / per damaged monster / per recipient |
| `X()`, `Y()` | `character_attack_common.go:360` (caster coords for reflect) | captured once per swing |
| `Equipment().Get("weapon")` → `TemplateId()` | projectile `:89,92,132,141,188-192`; writer `:34-38` | per swing / per recipient |
| `Inventory().Consumable().Assets()` | projectile `:123`; writer `:54` | per swing / per recipient |
| `Inventory().Cash().Assets()` | writer `:47` (cash bullets) | per recipient |

The equipment map is *derived*: `Model.SetInventory` rebuilds it from the equipable compartment's negative slots (`character/model.go:284-324`) — so weapon-slot correctness follows from asset-slot correctness in the inventory component.

### 2.2 atlas-data skill effects — uncached REST on every lookup (`data/skill/processor.go:30-46`)

| Read | Fields | Frequency |
|---|---|---|
| Attack skill `se` (`character_attack_common.go:292`) | `MonsterStatus(), Duration(), HPConsume(), MPConsume(), BulletConsume()` | 1 GET per skill swing |
| MP-Eater effect (`character_attack_common.go:231`) | `Prop(), X()` | 1 GET **per damaged monster** (magic) |
| Writer mastery (`socket/writer/character_attack_common.go:214`) | mastery from skill data | 1 GET **per broadcast recipient per swing** |

Correction to the PRD's §1 table: `se.Prop()/X()` for MP Eater come from a separately-fetched passive effect, not the attack `se`; and the mastery lookup is per-recipient (the body producer runs inside `session.Announce` per recipient, `session/processor.go:191`), making it the largest hidden fan-out on the path.

### 2.3 Monster — conditional REST `GET {MONSTERS}/monsters/{uniqueId}` (`monster/requests.go:30-32`)

Reflect branch `character_attack_common.go:134` (only when the local `StatusMirror` reports a matching reflect — `:133`): reads `X(), Y(), MonsterId()`. MP Eater `character_attack_common.go:240` (per damaged monster, magic skill attacks): reads `MaxMp(), Mp()`. These are the duplicate FR-4.2 targets.

### 2.4 Buffs — `GET {BUFFS}/characters/{id}/buffs` once per **ranged** swing (`character_attack_projectile.go:97`, `character/buff/requests.go:16-18`): reads `Expired()`, `Changes()[].Type()` for SOUL_ARROW (skip consumption, `:107`) and SHADOW_PARTNER (double count, `:232`).

### 2.5 Effective stats — `GET {EFFECTIVE_STATS}/.../stats` lazily, ≤1 per swing, only when a VENOM apply occurs (`character_attack_common.go:327-339`): reads `Luck`, `MagicAttack`.

### 2.6 Already local (unchanged): `monster.StatusMirror.GetReflect` (`monster/status_mirror.go:222`), session fields (`s.CharacterId()`, `s.Field()`, `s.WorldId()`, `s.ChannelId()`), tenant from context.

## 3. Alternatives considered

**A. Composed-model snapshot (chosen).** Store the three REST-shaped components; compose the exact decorated `character.Model` on read. Pros: `processAttack`, the projectile planner, and all four writers consume the identical type — FR-4.6 wire equivalence reduces to "the composed model equals the fetched model"; per-component invalidation maps 1:1 onto the three underlying GETs; reuse (FR-3.5) is natural because the API returns the type every handler already understands. Cons: holds more state than the attack path strictly needs (full inventory vs. weapon+projectiles). Accepted: memory is session-bounded, and the FR-2 audit showed inventory events are rich enough to maintain the full model — a narrower projection would save memory but not accuracy risk.

**B. Narrow attack-projection snapshot** (level/job/x/y + weapon + projectile assets + skill-id→level map). Pros: minimal state. Cons: the planner and writers take `character.Model`, so this needs an adapter constructing partial models — a fresh source of wire divergence; every future adopter (movement, chat, use-item) needs new fields anyway, defeating FR-3.5. Rejected.

**C. Short-TTL cache of the decorated character model, no events.** Pros: trivial. Cons: serves known-stale data on the path that decides damage application and projectile consumption; PRD prohibits caching over identified staleness (FR-2.4). Rejected.

**D. Redis-shared snapshot across pods.** Rejected by PRD non-goal (in-process per-pod only, task-120 precedent).

## 4. Snapshot architecture (FR-3)

### 4.1 Package and state

New package `services/atlas-channel/atlas.com/channel/character/snapshot/`:

```go
// registry.go — singleton per project cache discipline (sync.Once + RWMutex,
// tenant-partitioned; see .claude/skills/backend-dev-guidelines/resources/patterns-cache.md)
type entry struct {
    core      character.Model   // undecorated base model
    coreGen   uint64            // generation counter; bumped on invalidate
    coreValid bool
    skills      []skill.Model
    skillsGen   uint64
    skillsValid bool
    inv      inventory.Model
    invGen   uint64
    invValid bool
    // position overlay — locally fed, independent validity
    posX, posY int16
    posValid   bool
    // cached composition; rebuilt when any component changes
    composed      character.Model
    composedValid bool
}
type Registry struct {
    mu        sync.RWMutex
    perTenant map[uuid.UUID]map[uint32]*entry // tenant -> characterId -> entry
}
```

### 4.2 Read API (FR-3.5 — shaped for reuse)

`snapshot.NewProcessor(l, ctx)` exposes:

- `Get(characterId uint32) (character.Model, error)` — the one call the attack path makes. Fast path: all components valid → return the cached composed model (RLock, value copy). Slow path: for each invalid/missing component, run **only that component's** existing REST provider (base `requests.ById`, `inventory.GetByCharacterId`, skills provider — the same code `GetById` + decorators use today), then backfill under a generation check and recompose. Position: if `posValid`, overlay X/Y onto the composed model; else use the base model's X/Y from REST (exactly today's source).
- `BuffsProvider(characterId uint32) model.Provider[[]buff.Model]` — the projectile gate's read (see §7). Kept as a separate provider because buffs are not part of `character.Model`.
- `Evict(characterId uint32)` / `EvictTenant(tid uuid.UUID)`.
- Internal mutators used by consumers/handlers: `ApplyStatValues`, `SetLevel`, `SetExperience`, `UpsertSkill`, `RemoveSkill`, `ApplyAsset…`, `InvalidateCore/Skills/Inventory`, `SetPosition`, `UpsertBuff`, `RemoveBuff` — all **update-only**: they no-op when no entry exists (events never create entries; task-120 discipline, required because consumers run from `LastOffset`).

A REST fallback failure propagates exactly as today's error path (FR-3.4): `Get` returns the error; it never serves a stale component in place of a failed refetch.

### 4.3 Lifecycle (FR-3.2 — resolves PRD Open Question 3: lazy)

- **Populate lazily** on first `Get` miss (the first swing pays today's 3 GETs once). Attach-time population would race post-login mutations and add login-path work for characters who never attack.
- **Evict on session destroy** — one hook in `session.Processor.Destroy` (`session/processor.go:330`, next to `getRegistry().Remove` at `:332`), which covers logout, disconnect, and channel change (all go through Destroy). Plus `snapshot.GetRegistry().EvictTenant` wired into the existing `listener.RegisterEvictor` block (`main.go:287-290`).
- Memory: O(concurrent local sessions) entries; no TTL needed.

### 4.4 Concurrency and the refetch race (FR-3.6, NFR-4, NFR-6)

Writers: Kafka consumer goroutines, the movement socket handler (position), lifecycle callbacks. Readers: socket-handler goroutines. `sync.RWMutex`, value-copy reads, no goroutines spawned.

The stale-backfill race (invalidation arrives while a REST refetch is in flight, then the older REST result overwrites the newer invalidation) is closed with the per-component **generation counter**: `Get` records `gen` before fetching; the backfill applies only if `gen` is unchanged, otherwise the component stays invalid and the *fetched value is still returned to this caller* (it is exactly what REST would have returned today — no worse), while the next read refetches. In-place event updates carry absolute values only (stat `Values`, `QUANTITY_CHANGED` absolute quantity, MOVED set-slot-absolute by AssetId, LEVEL/EXPERIENCE `Current`, buff add/remove keyed by source) — idempotent under at-least-once redelivery.

## 5. Event maintenance (FR-3.3) — handler additions per component

All handlers are additive registrations on existing consumers (pattern: one `rf(topic, handler)` block per event type, e.g. `kafka/consumer/character/consumer.go:54-79`); existing handler behavior is untouched. Full audit with verdicts: `event-coverage.md`.

| Component | Event → snapshot action |
|---|---|
| core | STAT_CHANGED with complete `Values` → apply absolute values in place; without → `InvalidateCore`. LEVEL_CHANGED → set level. EXPERIENCE_CHANGED → set exp. MESO/FAME_CHANGED (deltas) → `InvalidateCore`. MAP_CHANGED → update field; set position from `TargetX/Y` when `UseTargetPosition`, else `posValid=false`. |
| skills | CREATED/UPDATED → upsert (level, masterLevel, expiration). DELETED (**new handler** — event exists, channel ignores it today) → remove. COOLDOWN_* → ignored (not in read-set, v1). |
| inventory | asset CREATED/UPDATED/ACCEPTED → full-asset upsert by AssetId. QUANTITY_CHANGED → set absolute quantity. MOVED → set slot absolute by AssetId. DELETED → remove by AssetId. RELEASED/EXPIRED (thin) → `InvalidateInventory`. compartment DELETED/CAPACITY_CHANGED (**new handlers**) → `InvalidateInventory`. MERGE_COMPLETE/SORT_COMPLETE → `InvalidateInventory` (bulk rearrangement; replay is fragile). RESERVED/RESERVATION_CANCELLED → no-op (REST parity — reservations aren't in asset rows; VERIFY-AT-EXECUTION, event-coverage.md §4). |
| buffs | APPLIED → upsert (temp-stat changes, ExpiresAt). EXPIRED → remove. Reads self-filter entries past `ExpiresAt` (bounds the atlas-buffs-restart residual, event-coverage.md §5). |
| position | Not an event: `movement.ForCharacter` (`movement/processor.go:46-59`) calls `SetPosition` with its folded final X/Y **synchronously, before emitting** the movement command — the freshest possible source (FR-2.5; today's REST value is this same fold after Kafka→Redis→nginx round-trips). |

### 5.1 The one producer change (FR-2.4(a)): STAT_CHANGED `Values`

atlas-character populates `Values` (absolute post-mutation values, keys per the convention of the four sites that already populate it — `character/processor.go:905,1408,1623,1867`) on **all** STAT_CHANGED emissions (~15 remaining call sites in `character/processor.go`, one service, producer-only). The field already exists with `omitempty` — additive and backward-compatible; existing consumers ignore it. Rollout-safe: snapshot handlers treat a nil/incomplete `Values` as `InvalidateCore`, so mixed versions degrade to invalidate-and-refetch, never to wrong data.

Rationale: `CHANGE_MP` fires on every skill swing with MP cost; without rich values the snapshot self-invalidates each swing and the steady state becomes 1 GET/swing rather than 0 (still a 3+N→1 win, but not the PRD's goal).

## 6. Skill-data TTL cache (FR-4.3)

`data/skill/cache.go`: in-process singleton, tenant→skillId→`cacheEntry{model, negative, expiresAt}`, fronting `data/skill.Processor.GetById` (so `GetEffect`'s level indexing is unchanged, `data/skill/processor.go:42-45`). Semantics copied from task-060's landed implementation (`services/atlas-monsters/atlas.com/monsters/monster/information/cache.go`): positive TTL 5m / negative 30s, negative caching only for `errors.Is(err, requests.ErrNotFound)`, transient errors never cached, `upstreamFn` test seam, lazy expiry on read. Env: `SKILL_DATA_CACHE_ENABLED` (default true), `SKILL_DATA_CACHE_TTL` (default 5m, clamp [1s,24h]), `SKILL_DATA_CACHE_NEGATIVE_TTL` (default 30s, clamp [0s,5m]). Note the landed task-060 code is Redis-backed; per PRD non-goal this is the **in-process** re-expression of its semantics (as task-120 §5.4 already designed for monster info) — there is no landed in-process exemplar to copy verbatim.

This one cache removes: the attack-`se` GET (per skill swing), the MP-Eater effect GET (per damaged monster), and the writer mastery GET (per broadcast recipient) — the writer needs no code change beyond its processor reading through the cache.

## 7. Attack-path adoption (FR-4)

- `character_attack_common.go:271-272` → `snapshot.NewProcessor(l, ctx).Get(s.CharacterId())`. Everything downstream (skill ownership check, planner, damage pipeline, writer closures) consumes the returned `character.Model` unchanged; writers receive the same already-fetched model as today (FR-4.5).
- **Monster dedup (FR-4.2)**: `processAttack` builds a per-swing memoized resolver `getMonster(monsterId)` backed by task-120's `LiveMirror.Lookup` with REST fallback+`Put` backfill; wire it into `deps.getMonster` *and* pass it to `mpEaterTryProc` (replacing its direct `mp.GetById`, `character_attack_common.go:240`). One resolve per damaged monster serves reflect and MP Eater regardless of hit or fallback. Lazy: entries with no reflect match and non-magic attacks still resolve nothing (§2.3 conditionality preserved).
- **Mirror extension (FR-5.1)**: `LiveEntry` gains `X, Y int16` — seeded from the monster CREATED handler's already-fetched model, updated synchronously from the monster-movement fold (same local-write pattern as character position), backfilled by the fallback `Put`. task-120 is docs-only at time of writing; per FR-5.2 this is designed against its committed plan and MUST be re-validated (entry shape, function names) at execution time. Sequencing: task-120 lands first (owner decision).
- **Projectile gate**: `ProjectileProcessorImpl.Plan`'s `bp.GetByCharacterId` (`character_attack_projectile.go:97`) → `snapshot.BuffsProvider` (event-maintained per §5; lazy REST seed on miss via the existing buff provider; failure keeps today's fail-open "assume no buffs" semantics).
- **Venom** (`character_attack_common.go:327-339`): unchanged — REST with per-swing memoization (event-coverage.md §6 verdict).
- `ChangeHP/ChangeMP` cost emissions: unchanged (commands, not reads).

Steady-state REST on the attack path after adoption: **zero**; every REST call remaining is a counted fallback.

## 8. Observability (FR-6) and shadow verification (resolves Open Question 5)

Prometheus per task-120's design (which introduces atlas-channel's first `promauto` + `promhttp` wiring at `/api/metrics` via `MountHandler`; if task-122 executes first, it brings the mount with it — same code either way):

- `atlas_channel_char_snapshot_reads_total{tenant, component, outcome="hit|miss|fallback_success|fallback_failure"}`
- `atlas_channel_char_snapshot_updates_total{tenant, component, kind="event_update|invalidation|backfill|backfill_discarded"}`
- `atlas_channel_skill_data_cache_total{tenant, outcome="hit|negative_hit|miss"}`
- Debug logs on every fallback with characterId + component (FR-6.2).

**Shadow verification: yes, env-gated sampling (diverging from task-121's "no").** task-121 declined shadow mode because it would keep alive REST plumbing the task deletes; here the REST providers *must* survive as the fallback path, so shadow costs nothing structurally. `CHAR_SNAPSHOT_SHADOW_SAMPLE_RATE` (float, default 0): on a sampled snapshot hit, asynchronously fetch the REST equivalent and compare the attack-relevant projection (level, job, x/y tolerance-banded, weapon templateId, consumable/cash asset quantities, skill id→level set, active projectile-gate buffs), logging Warn + `atlas_channel_char_snapshot_divergence_total{tenant, component}` on mismatch. Default-off in production; enabled in staging during the FR-8 soak. This directly answers the owner's accuracy concern with runtime evidence rather than test-time argument alone.

## 9. Testing (FR-7)

- **Snapshot unit tests**: hit path; per-component miss→fallback→backfill; every event type in §5's table (in-place value, invalidation, no-op-when-absent); generation check discards stale backfill; eviction on destroy; `-race` concurrent read/update/evict.
- **Wire equivalence (FR-4.6/7.2)**: extend the existing `damageInfoEntryDeps` seams (plus a planner/writer harness) — fixed `AttackInfo` + fixture models driven through both the REST-shaped model and the snapshot-composed model must produce byte-identical emitted commands and packets. Builder pattern only, no `*_testhelpers.go`.
- **Dedup test (FR-7.3)**: counting resolver proves one monster resolve per damaged monster serves reflect + MP Eater.
- **Staleness-window test (FR-7.4)**: skill UPDATED event between two attacks → second attack's snapshot read reflects the new level.
- **Cache tests**: TTL expiry, negative caching classification, kill-switch, seam-driven upstream counting.
- Producer change: atlas-character unit tests asserting `Values` present + absolute on each STAT_CHANGED site.

## 10. Execution-time verification checklist (carry into plan.md)

1. Reconcile with task-120 as actually landed (mirror API names, entry shape, metrics bootstrap) — FR-5.2.
2. VERIFY: `GET /characters/{id}/inventory` does not net out reservations (event-coverage.md §4); if it does, reservation events must join the inventory component.
3. VERIFY: `Values` key convention at the four existing populated sites before extending (`character/processor.go:905` et al.).
4. VERIFY: MAP_CHANGED `UseTargetPosition=false` path — confirm reflect-before-first-move falls back to REST position correctly.
5. Escalate separately to the owner: `RequestReserve` first-request-only loop (`atlas-inventory .../compartment/processor.go:767`) — candidate pre-existing bug, not part of this task.

## 11. Risks

| Risk | Mitigation |
|---|---|
| Event races vs REST backfill | Generation counters; absolute-value-only in-place updates (idempotent) |
| Mixed-version rollout (old STAT_CHANGED without Values) | Nil/incomplete Values → invalidate (correct, just slower) until atlas-character rolls |
| atlas-buffs pod restart drops buffs without EXPIRED | Snapshot self-expires at event `ExpiresAt`; divergence bounded by buff duration (event-coverage.md §5) |
| task-120 plan drift | FR-5.2 re-validation step; task-120 lands first |
| Silent projection drift in production | Shadow sampling + per-component divergence counter; >95% hit-rate target (NFR-7) verifiable at `/api/metrics` |
| Snapshot outliving its session | Eviction inside `session.Destroy` itself (not a parallel code path), plus tenant evictor |
