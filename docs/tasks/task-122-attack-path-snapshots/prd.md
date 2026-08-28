# Attack-Path Snapshots (PS-1) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-02
---

## 1. Overview

Every attack packet handled by atlas-channel pays a fan-out of synchronous REST calls through nginx before any damage is applied or broadcast. `processAttack` (`services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:266-364`) is the entry point for all four attack types (melee/ranged/magic/energy), and it is the hottest game-loop path after movement: combat latency and nginx/service load scale with fight intensity. This is finding **PS-1 (Critical)** in `docs/architectural-improvements.md` (2026-07-02 review).

The finding's "3+2N" count actually understates the fan-out. The full per-swing REST inventory, verified against the code:

| # | Call | Site | Frequency | Target service |
|---|------|------|-----------|----------------|
| 1 | `character.GetById` | `character_attack_common.go:272` | every swing | atlas-characters |
| 2 | `InventoryDecorator` (`inventory.GetByCharacterId`) | `character/processor.go:75` | every swing | inventory/compartment services |
| 3 | `SkillModelDecorator` | `character/processor.go` | every swing | atlas-skills |
| 4 | `skill.GetEffect` | `character_attack_common.go:292` | every swing with a skill | atlas-data |
| 5 | `buff.GetByCharacterId` (projectile gate) | `character_attack_projectile.go:99` | every ranged swing | atlas-buffs |
| 6 | `monster.GetById` (reflect) | `character_attack_common.go:134` | per damaged monster with matching reflect | atlas-monsters |
| 7 | `monster.GetById` (MP Eater — duplicate of #6) | `character_attack_common.go:240` | per damaged monster, magic attacks | atlas-monsters |
| 8 | `skill.GetEffect` (MP Eater effect) | `character_attack_common.go:231` | per damaged monster, magic attacks | atlas-data |
| 9 | `effective_stats.GetByCharacterId` (venom) | `character_attack_common.go:332` | lazily, once per attack that applies VENOM | atlas-query-aggregator/effective-stats |
| 10 | mastery skill-effect lookups inside `computeMasteryForWeapon` | `socket/writer/character_attack_common.go:41` | per broadcast encode | atlas-data |

All calls are blocking and routed through nginx. A multi-target magic attack against 6 mobs can exceed 20 REST round-trips before the swing is fully processed.

This task removes REST from the steady-state attack path by (a) introducing a **session-scoped character snapshot** (character core + skills + inventory/equipment) maintained from Kafka events atlas-channel already consumes, (b) adopting the **task-120 live-monster mirror** for the reflect/MP-Eater monster reads and eliminating the duplicate fetch, and (c) applying the **task-060/task-120 in-process TTL cache pattern** to immutable atlas-data skill-effect lookups. REST remains only as a miss fallback that backfills the local state.

**The central risk of this task is correctness, not mechanics.** Unlike task-120 (monster state, one owning service, one event topic) and task-121 (session field, locally owned), the character snapshot aggregates state owned by *multiple* services — atlas-characters, atlas-skills, the inventory/compartment/asset services, atlas-buffs — each mutated by yet more services (sagas, consumables, NPC shops, trades, drops, quests, GM commands). A snapshot that misses one mutation path serves stale data on the path that decides damage application, projectile consumption, and skill ownership checks. The owner's explicit direction (2026-07-02): put critical thought into cross-service accuracy. This PRD therefore makes a **complete mutation-source/event-coverage audit a prerequisite functional requirement** (FR-2), and makes per-component inclusion in the snapshot conditional on that audit — a component whose mutations cannot be reliably observed stays on REST (or gets its event gap fixed) rather than being cached wrongly.

## 2. Goals

Primary goals:

- Zero REST calls on the steady-state attack path: character/skill/inventory reads served from a session-scoped snapshot, monster reads from the task-120 mirror, skill-effect reads from an in-process TTL cache, effective-stats and buff reads per the audit's verdict (snapshot, cache, or retained REST — decided by evidence, not convenience).
- REST retained only as **miss fallback** (cold start, event races, audit-identified gaps), with fallback hits backfilling local state.
- A per-component mutation-source audit proving, with file:line evidence, that every mutation of snapshotted state emits an event atlas-channel observes — or explicitly documenting the gap and its mitigation.
- The snapshot is **shaped for reuse** (owner decision): a clean read API that other hot handlers (movement, chat, use-item) can adopt in follow-up tasks, though only the attack path adopts it here.
- Observability: hit/miss/fallback/staleness counters per component, tenant-scoped, mirroring task-120's metrics requirement.
- No wire-level behavior change: packets emitted (ack, broadcast, damage, status) are byte-identical to today for identical inputs.

Non-goals:

- **PS-2 (task-121)** and **PS-3 (task-120)** — in flight; this task **depends on task-120 landing first** (owner decision) and adopts its mirror rather than re-implementing monster reads.
- No adoption of the snapshot in non-attack handlers (movement, chat, use-item, etc.) — the snapshot is built reusable, adoption elsewhere is follow-up work.
- No changes to the REST APIs of atlas-characters, atlas-skills, inventory/compartment services, atlas-buffs, or atlas-data. (Adding a missing *Kafka event emission* in an owning service is in scope if — and only if — the FR-2 audit proves a gap; see FR-2.4.)
- No Redis-backed caching; all local state is in-process, per-pod (task-120 precedent, owner decision there).
- No changes to damage formulas, reflect semantics, MP Eater semantics, or projectile consumption semantics.

## 3. User Stories

- As a player, I want attack packets processed without a fan-out of cross-service HTTP round-trips so that combat stays responsive during intense fights and mass mobbing.
- As a player, I want damage, projectile consumption, and skill checks computed from accurate state so that caching never causes wrong damage application, wrongly rejected attacks, or item desync.
- As an operator, I want nginx and character/skill/inventory service load decoupled from combat packet rate so that infrastructure load reflects state changes, not swings-per-second.
- As an operator, I want per-component hit/miss/fallback metrics so I can verify the snapshot is actually serving the hot path and detect projection drift before players do.
- As a developer, I want a single session-scoped snapshot API so future hot-path features read local state instead of each re-inventing REST calls.

## 4. Functional Requirements

### 4.1 Attack-path read-set inventory (prerequisite)

- **FR-1.1** Document, with file:line evidence, every field the attack path reads from each REST-sourced model. Known set (to be verified/extended during design):
  - `character.Model`: `Id()`, `Level()`, `JobId()`, `X()`, `Y()`, `Skills()` (id + level), `Inventory()` (cash + consumable assets for bullet resolution and projectile planning), `Equipment()` (weapon slot for mastery and projectile classification).
  - `monster.Model`: `MonsterId()`, `X()`, `Y()` (reflect), `MaxMp()`, `Mp()` (MP Eater).
  - `effect.Model` (atlas-data): full effect for the attacking skill; `Prop()`/`X()` for MP Eater's passive; mastery skill effects in the writer.
  - `buff` models: active temporary stats for the projectile gate (Soul Arrow, Shadow Partner) .
  - `effective_stats.RestModel`: `Luck`, `MagicAttack` (venom snapshot).
- **FR-1.2** The read-set inventory is committed to the task folder (`read-set.md` or a design.md section) so the snapshot's shape is derived from demonstrated need, not from mirroring entire REST models.

### 4.2 Mutation-source / event-coverage audit (prerequisite — the accuracy gate)

- **FR-2.1** For each snapshot component (character core stats/position, skills, inventory/equipment), enumerate every mutation path across the fleet: which services mutate the state, via which commands/sagas, and which Kafka event each mutation emits. Evidence is file:line in the owning service's producer and atlas-channel's consumer (`kafka/consumer/{character,skill,compartment,buff,...}`).
- **FR-2.2** For each mutation path, classify event coverage:
  - **Covered** — an event atlas-channel already consumes carries enough data to update the snapshot (e.g. character `STAT_CHANGED`/`LEVEL_CHANGED`, skill status events incl. cooldown/level, compartment `CREATED`/`DELETED`/`CAPACITY_CHANGED`/merge/sort/reservation events, buff applied/expired).
  - **Covered-but-thin** — an event signals *that* a change happened but not the new value; snapshot handles it by invalidate-and-refetch (targeted REST on next read), not by guessing.
  - **Gap** — a mutation with no observable event.
- **FR-2.3** The audit result is a committed artifact (`event-coverage.md`) in the task folder: component × mutation-path × event × handling. This is the document the design phase and reviewers verify against; it is the deliverable that answers the owner's accuracy concern.
- **FR-2.4** Every **Gap** gets an explicit disposition, decided in design with owner visibility: (a) add the missing event emission in the owning service (small, bounded producer change — in scope), (b) exclude that component from the snapshot and keep its REST read, or (c) demote the component to invalidate-on-any-signal + short TTL. Silently caching over a known gap is prohibited.
- **FR-2.5** Character **position** gets explicit treatment: `c.X()`/`c.Y()` feed the reflect bounds check. The audit must establish where authoritative position lives from atlas-channel's perspective (locally observed movement packets vs. atlas-characters' async projection) and the design must pick the source accordingly. Note today's REST read is itself an async projection of movement events, so the snapshot must be *no worse*, not perfect.

### 4.3 Session-scoped character snapshot

- **FR-3.1** A session-scoped, tenant-scoped, per-pod in-memory snapshot holding the audited read-set (character core, skills, inventory/equipment as scoped by FR-2's verdicts), keyed by character id, following the established registry discipline (`sync.Once` singleton, `sync.RWMutex`).
- **FR-3.2** Lifecycle: populated on first need (lazy, via the existing REST calls) or at session attach — design-phase choice; **evicted on session destroy/logout/channel-change** so memory is bounded by concurrent local players and no snapshot outlives its session.
- **FR-3.3** Maintained from the Kafka events identified in FR-2, as additions alongside existing consumer handlers (existing handler behavior unchanged). Thin events trigger invalidation (next read refetches that component via REST and backfills); rich events update in place.
- **FR-3.4** On any component miss or invalidated component, the read falls back to the existing REST call for that component only, backfills the snapshot, and counts the fallback (FR-6). A REST fallback failure surfaces exactly as today's error path — the snapshot must never convert a hard failure into stale-success.
- **FR-3.5** The read API is **shaped for reuse**: exposed as a processor/provider that any handler can call with a character id, returning the composed snapshot model — not a private helper of the attack file. Adoption beyond the attack path is out of scope, but the API must not need redesign for it.
- **FR-3.6** Concurrency: safe under concurrent socket-handler goroutines (multiple packets from one client), Kafka consumer goroutines, and session-lifecycle callbacks; `go test -race` clean.

### 4.4 Attack-path adoption

- **FR-4.1** `processAttack` resolves character + skills + inventory from the snapshot instead of `cp.GetById(cp.InventoryDecorator, cp.SkillModelDecorator)`.
- **FR-4.2** The reflect monster read (`:134`) and MP Eater monster read (`:240`) are served by one shared read from the task-120 live-monster mirror per damaged monster — the duplicate fetch is eliminated regardless of mirror hit or REST fallback.
- **FR-4.3** `skill.GetEffect` calls (attack skill at `:292`, MP Eater at `:231`, mastery lookups in the writer) are served from an in-process TTL cache for atlas-data skill effects, following the task-060/task-120 template-cache pattern (TTL expiry, no event invalidation, negative-result handling per that precedent).
- **FR-4.4** The venom `effective_stats` read and the projectile-gate `buff` read are resolved per the FR-2 audit verdict: snapshot component, event-maintained, or retained-REST with the existing per-attack memoization. The decision and its rationale are recorded in design.md.
- **FR-4.5** The broadcast writers (`CharacterAttack{Melee,Ranged,Magic,Energy}Body` / `preComputeAttackValues`) consume the same already-fetched snapshot model passed in as `c` today — no writer-side REST reintroduced.
- **FR-4.6** Wire-level behavior is unchanged: for identical inputs, damage application commands, status applications, reflect emissions, projectile consumption emissions, and broadcast packet bytes are identical to today.

### 4.5 Dependency on task-120

- **FR-5.1** This task assumes task-120 (live-monster mirror + atlas-data TTL cache in atlas-channel) has landed (owner decision). The attack path consumes the mirror's read API as-is; if adoption reveals a missing field (e.g. monster `X()`/`Y()` for reflect — task-120's FR-1.2 set is field/monsterId/MP/aggro), extending the mirror entry is in scope for this task.
- **FR-5.2** If task-122 reaches design before task-120 merges, the design must be written against task-120's committed plan and re-validated at execution time.

### 4.6 Observability

- **FR-6.1** Tenant-scoped counters per snapshot component: hit, miss, REST fallback, event-driven update, invalidation. Same mechanism/style as task-120's mirror metrics.
- **FR-6.2** Log-level visibility (Debug) for fallbacks on the attack path so drift is diagnosable per character without metric cardinality explosion.

### 4.7 Testing

- **FR-7.1** Unit tests cover: snapshot hit path, per-component miss + fallback + backfill, event-driven update for each covered event type, invalidate-and-refetch for thin events, eviction on session destroy, and concurrent read/update safety (race detector).
- **FR-7.2** Attack-path tests prove FR-4.6 equivalence: for a fixed input set, the emitted commands/packets with the snapshot are identical to the pre-change behavior (extend the existing `damageInfoEntryDeps`-style seams; project Builder pattern, no `*_testhelpers.go`).
- **FR-7.3** A test demonstrates the duplicate-monster-fetch elimination (one monster read per damaged monster serves both reflect and MP Eater).
- **FR-7.4** A staleness-window test: a skill-level change event arriving between two attacks is reflected in the second attack's snapshot read.

## 5. API Surface

No new or modified external REST endpoints. All existing REST endpoints remain (they are the fallback path and serve other consumers). New internal Go surface in atlas-channel: the snapshot processor/provider (FR-3.5) and any mirror-entry extension in the task-120 mirror (FR-5.1). Kafka contract additions only if FR-2.4(a) dispositions require an owning service to emit an event it should already have been emitting; any such addition is a new event type or field on an existing status topic, additive and backward-compatible.

## 6. Data Model

No persistent data model changes. All new state is in-memory, per-pod, tenant-scoped, session-bounded (FR-3.2), never shared cross-pod, never persisted. Memory envelope: O(concurrent local sessions) snapshot entries + the task-120 caches; eviction on session destroy keeps it bounded.

## 7. Service Impact

- **atlas-channel** — the only service with certain code changes:
  - New snapshot registry/processor + Kafka handler additions in existing consumers (`character`, `skill`, `compartment`, `buff` as scoped by FR-2).
  - `socket/handler/character_attack_common.go`, `character_attack_projectile.go`, `socket/writer/character_attack_common.go`: read-path swap (FR-4).
  - Possible small extension to the task-120 mirror entry (FR-5.1).
- **atlas-characters / atlas-skills / inventory-compartment services / atlas-buffs** — no REST/API changes; steady-state read load from combat drops to near zero. Possible bounded producer additions only per FR-2.4(a) audit verdicts.
- **atlas-data / atlas-monsters** — no changes (task-120 already covers their caching); combat-driven read load drops.
- **nginx** — hot-path request volume drops by the full table-in-§1 inventory; no config change.

## 8. Non-Functional Requirements

- **NFR-1 Performance.** Steady-state attack processing performs zero network I/O for state reads; snapshot reads are lock-bounded in-process lookups. Target: state-read cost negligible relative to packet decode/encode.
- **NFR-2 Availability.** Combat no longer has a runtime dependency on nginx/atlas-characters/atlas-skills/inventory services for steady-state swings; an outage degrades only cold-start/fallback paths.
- **NFR-3 Consistency/staleness.** Snapshot consistency equals Kafka event propagation lag, accepted by the owner (2026-07-02) with the observation that today's REST reads are themselves async projections of the same events — the snapshot must be demonstrably no staler than that baseline for every covered component (the FR-2 audit is the demonstration). Where a component cannot meet this bar, it stays on REST (FR-2.4).
- **NFR-4 Correctness under redelivery.** Event handlers updating the snapshot must be idempotent (at-least-once delivery is normal post-migration); redelivered events must not corrupt counts/levels (prefer absolute-state events or invalidation over increments).
- **NFR-5 Multi-tenancy.** All snapshot state tenant-scoped via `tenant.MustFromContext`; no cross-tenant reads in enumeration or invalidation.
- **NFR-6 Concurrency.** Existing registry `RWMutex` discipline; no unbounded goroutines; `go test -race` clean across the module.
- **NFR-7 Observability.** FR-6 metrics sufficient to verify a >95% steady-state snapshot hit rate in staging combat, mirroring task-120's measurability goal.

## 9. Open Questions

1. **Position source (FR-2.5).** Serve reflect's attacker position from locally-observed movement state, from the snapshot fed by movement events, or continue reading atlas-characters for it? Resolution needs the audit's answer on where position is freshest from this pod's perspective.
2. **Buffs and effective-stats disposition (FR-4.4).** Both have candidate event feeds (buff applied/expired is already consumed); whether they join the snapshot now or stay REST-with-memoization is an audit-verdict decision.
3. **Populate-at-attach vs. lazy-populate (FR-3.2).** Attach-time population makes the first swing fast but adds login-path work and races with post-login mutations; lazy makes the first swing pay today's cost once. Default leaning: lazy.
4. **Snapshot granularity of inventory.** Full inventory model vs. only the attack-relevant projections (equipped weapon, cash/consumable assets). Smaller is easier to keep accurate; the read-set inventory (FR-1) decides.
5. **Shadow verification.** Sampled comparison (snapshot vs. REST result, log on divergence) carried through staging before trusting the snapshot, or FR-7 tests + staging playtest sufficient? task-121 raised the same question; align the answer across the three game-loop tasks.

## 10. Acceptance Criteria

- [ ] `read-set.md` (or design.md section) documents every attack-path field read with file:line evidence (FR-1).
- [ ] `event-coverage.md` audit exists: component × mutation path × event × handling classification, with file:line evidence in owning-service producers and atlas-channel consumers; every Gap has an owner-visible disposition (FR-2).
- [ ] Session-scoped snapshot implemented per FR-3: lazy/attach population, event-maintained, invalidate-and-refetch for thin events, per-component REST fallback with backfill, eviction on session destroy, reusable read API.
- [ ] `processAttack` performs zero REST calls on the steady-state path: character/skills/inventory from snapshot, monster reads from the task-120 mirror (one read per damaged monster — duplicate eliminated), skill effects from TTL cache, venom/buff reads per audit verdict (FR-4).
- [ ] Broadcast writers consume the snapshot model with no writer-side REST (FR-4.5).
- [ ] Wire-level equivalence tests pass: identical commands/packets for identical inputs vs. pre-change behavior (FR-4.6, FR-7.2).
- [ ] Hit/miss/fallback/update/invalidation counters exposed per component, tenant-scoped (FR-6).
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in atlas-channel (and any owning service touched per FR-2.4a); `docker buildx bake atlas-channel` (and any touched service) succeeds; `tools/redis-key-guard.sh` clean.
- [ ] No `// TODO`, stubs, or deferred-but-producible work in landed commits.
