# Monster Movement Local State (PS-3) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-02
---

## 1. Overview

Every monster move packet handled by atlas-channel currently pays synchronous REST round-trips through nginx before it can ack the controller and broadcast to observers. `movement/processor.go` (`ForMonster`) calls `monster.GetById(objectId)` — a REST call to atlas-monsters — on **every** move packet, to obtain live state it uses for a field-consistency check, the monster's template id, its current MP (for `ackMp` forecasting), and its controller-aggro flag. On basic-attack moves it additionally calls `monsterinfo.GetById(mo.MonsterId())` — a REST call to atlas-data — to fetch immutable per-template attack info (`conMP`).

Monster movement is one of the highest-frequency packet types in the game loop, so this is finding **PS-3 (High)** in `docs/architectural-improvements.md` (2026-07-02 review): per-move REST amplification through nginx on a hot path, adding latency and a shared point of failure to real-time gameplay. (Note: the finding text says both calls happen per move; in reality the atlas-data call fires only on basic-attack moves — the live-state call to atlas-monsters is the dominant, truly per-move cost. This PRD reflects the actual code.)

Both calls are eliminable with data atlas-channel already receives or can cache:

- **Live state** — atlas-channel already consumes the `monster_status_event` topic and maintains an in-memory per-pod projection for status effects (`monster/status_mirror.go`) and predicted skills (`monster/inbox.go`). The event envelope carries `UniqueId`, `MonsterId`, and full field coordinates (world/channel/map/instance) on every event, and dedicated events exist for creation, destruction, kill, control handoff, aggro changes, and MP changes. This task extends that established pattern into a **live-monster mirror**: an in-memory, tenant-scoped projection keyed by monster object id, holding exactly what the movement path reads. On a mirror miss, fall back to the existing REST call and backfill the mirror.
- **Template data** — the atlas-data payload is immutable per id between atlas-data deployments. task-060 already built a TTL cache for this exact lookup inside atlas-monsters and explicitly deferred atlas-channel's `monster/information` package as a follow-up; this task is that follow-up, implemented as an **in-process TTL cache** (per user decision — no new Redis hop on the hot path).

## 2. Goals

Primary goals:

- Zero REST calls on the steady-state monster movement path: `ForMonster` resolves live monster state from the in-process mirror and template attack info from the in-process TTL cache.
- REST is retained only as a **miss fallback** (cold start after pod restart, event/packet races), and a fallback hit backfills the mirror so subsequent moves are local.
- The mirror is reusable: shaped so the PS-1 attack-path callers (`MP Eater`, reflect monster fetches) can adopt it in a later task without redesign.
- Observability: hit/miss/fallback counters for the mirror and hit/miss/negative-hit counters for the template cache, tenant-scoped, so the >95% steady-state hit-rate claim is measurable.
- No wire-level behavior change: ack packets, broadcast packets, and Kafka movement commands are byte-identical to today for identical inputs.

Non-goals:

- **PS-2** — the `ForOtherSessionsInMap` broadcast REST call to atlas-maps inside the same function stays as-is. Out of scope.
- **PS-1** — attack-path REST fan-out (character/inventory/skills snapshots, per-monster fetches). Out of scope; this task only leaves the mirror ready for it.
- **PS-4** — Redis optimistic-lock costs inside atlas-monsters. Out of scope.
- No changes to atlas-data (no new endpoints, no cache headers).
- No Redis-backed caching in atlas-channel for either data set (in-process only, per user decision).
- No event-driven invalidation of the template cache (TTL expiry only, matching task-060's v1 scope).
- No changes to other `monster.GetById` callers in atlas-channel beyond the movement path.

## 3. User Stories

- As a player, I want monster movement to be acked and broadcast without cross-service REST latency so that mobs move smoothly even when nginx or atlas-monsters is under load.
- As an operator, I want the per-move REST amplification (2 calls × move rate × mob count) removed so that nginx and atlas-monsters load no longer scale with mob movement volume.
- As a developer working on PS-1, I want a live-monster mirror with a clean read API so the attack path can drop its per-monster REST fetches later.
- As an operator, I want mirror/cache hit-rate metrics so I can verify the fallback path is rare and detect projection drift.

## 4. Functional Requirements

### 4.1 Live-monster mirror (atlas-channel)

- **FR-1.1** A new in-memory, per-pod, tenant-scoped mirror in `atlas-channel/monster` keyed by monster object id (`uniqueId`), following the `StatusMirror` precedent (singleton via `sync.Once`, `sync.RWMutex`, per-tenant nesting).
- **FR-1.2** Each entry holds at minimum: field identity (worldId, channelId, mapId, instance), `monsterId` (template id), current MP, and `controllerHasAggro` — the exact set `ForMonster` reads today (`processor.go:112-153`).
- **FR-1.3** The mirror is populated/updated from the already-consumed `monster_status_event` topic:
  - `CREATED` → insert entry (envelope carries field + monsterId; initial MP sourcing is a design-phase decision — see Open Questions).
  - `DESTROYED`, `KILLED` → remove entry.
  - `AGGRO_CHANGED`, `START_CONTROL` → update `controllerHasAggro`.
  - `MP_CHANGED` → update MP.
  - Event handlers must be additions alongside the existing packet-emitting handlers, not replacements; existing handler behavior is unchanged.
- **FR-1.4** Mirror mutations and reads are safe under concurrent access from Kafka consumer goroutines and socket-handler goroutines.
- **FR-1.5** Entries are bounded by live-monster population; `DESTROYED`/`KILLED` eviction plus a defensive staleness sweep (design-phase detail) prevent unbounded growth across map churn.

### 4.2 Movement path consumption

- **FR-2.1** `movement.Processor.ForMonster` resolves the monster via the mirror instead of `monster.NewProcessor(...).GetById(objectId)`.
- **FR-2.2** On mirror miss, `ForMonster` falls back to the existing REST `GetById`, and on success backfills the mirror entry so subsequent moves for that monster are local. On REST failure the current error behavior is preserved (log + return error).
- **FR-2.3** The field-consistency check (`f` vs. mirror entry's world/channel/map) and its rejection behavior are preserved unchanged.
- **FR-2.4** `ackMp` forecasting and the `useSkills`/aggro logic produce the same values as today given equivalent state.

### 4.3 Template info cache

- **FR-3.1** `atlas-channel/monster/information.GetById` is fronted by an in-process, tenant-scoped TTL cache: positive entries with configurable TTL (default 5 minutes), negative entries (fetch error / not found) with configurable shorter TTL (default 30 seconds) — mirroring task-060's defaults.
- **FR-3.2** The cache is transparent to callers: same method signature, no call-site changes beyond the movement path already using it.
- **FR-3.3** TTLs are configurable via environment variables with the defaults above.

### 4.4 Observability

- **FR-4.1** Counters (tenant-scoped): mirror hit, mirror miss, fallback success, fallback failure; template-cache hit, miss, negative-hit.
- **FR-4.2** Mirror misses on the movement path log at debug level with object id, so drift is diagnosable without log spam.

## 5. API Surface

No new or modified REST endpoints. No new Kafka topics. Existing REST requests (`atlas-monsters GET /monsters/{uniqueId}`, `atlas-data GET /data/monsters/{id}`) are demoted to fallback/cache-fill paths.

Possible (design-phase-gated) change: enrichment of an existing `monster_status_event` body emitted by atlas-monsters if MP event coverage proves insufficient (see Open Questions). Any such change must be additive (new fields only) so mixed-version deploys remain compatible.

## 6. Data Model

No database changes. All new state is in-process memory in atlas-channel:

- Live-monster mirror: `tenant → uniqueId → {field, monsterId, mp, controllerHasAggro, updatedAt}`.
- Template cache: `tenant → monsterId → {model | negative, expiresAt}`.

## 7. Service Impact

- **atlas-channel** (primary): new mirror + registration of its event handlers on the existing `monster_status_event` consumer; TTL cache in `monster/information`; `movement/processor.go` consumption changes; metrics.
- **atlas-monsters** (conditional): only if the design phase finds MP mutations not covered by `MP_CHANGED` events (e.g. skill-cast MP spend) — then additive event emission/enrichment to close the gap.
- No other services change.

## 8. Non-Functional Requirements

- **Performance:** steady-state monster move handling performs no REST calls; mirror/cache reads are lock-cheap (RWMutex read path).
- **Multi-tenancy:** all mirror/cache state is tenant-keyed (per `StatusMirror` precedent); one tenant's monsters never resolve from another tenant's entries.
- **Consistency:** a move packet may race its monster's `CREATED` event (consumer lag) — the REST fallback covers this, so correctness never depends on projection freshness. Consumer uses `kafka.LastOffset`, so a restarted pod starts with an empty mirror and warms via fallback.
- **Memory:** mirror size is O(live monsters on the channel); template cache is O(distinct monster templates) with TTL expiry.
- **No behavior drift:** wire packets and emitted Kafka movement commands are unchanged for identical logical state.
- **Testing:** Builder-pattern test setup (no `*_testhelpers.go`); unit tests cover mirror projection transitions, miss/fallback/backfill, TTL expiry (positive + negative), tenant isolation, and concurrency (`go test -race`).

## 9. Open Questions

1. **Initial MP at CREATED:** `StatusEventCreatedBody` carries only `ActorId`; the envelope has no MP. Options: (a) source max MP from the (cached) template on first need, (b) additively enrich the CREATED body with MP from atlas-monsters, (c) leave MP unset and let the first movement's fallback populate it. Resolve in design phase.
2. **MP_CHANGED coverage:** confirm atlas-monsters emits `MP_CHANGED` for every MP mutation the movement ack can observe (skill casts, basic-attack conMP spend, MP Eater drain). Known: `DrainMpCommandBody` documents MP_CHANGED emission for MP Eater. If gaps exist, additive enrichment in atlas-monsters is in scope (user-approved).
3. **Staleness sweep policy for mirror entries** (defensive eviction interval/threshold) — design-phase detail.

## 10. Acceptance Criteria

- [ ] Steady-state monster move packet (mirror warm, template cached) issues **zero** REST calls from `ForMonster` (verified by test seams/counters, not inspection).
- [ ] Mirror miss triggers exactly one REST fallback, backfills the entry, and the next move for that monster is REST-free.
- [ ] `DESTROYED`/`KILLED` events evict the mirror entry; a post-death move packet takes the fallback path and preserves today's error behavior when the REST lookup fails.
- [ ] Template info fetch on basic-attack moves is served from the in-process cache after first fetch; negative results are cached with the shorter TTL; TTLs configurable via env.
- [ ] `ackMp`, `useSkills`, field-consistency rejection, ack/broadcast packet bytes, and movement Kafka commands are unchanged for identical logical state (regression-covered).
- [ ] All mirror/cache state is tenant-scoped with a test proving cross-tenant isolation.
- [ ] Hit/miss/fallback metrics exposed for both mirror and template cache.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in atlas-channel (and atlas-monsters if touched); `docker buildx bake atlas-channel` (and `atlas-monsters` if touched) succeeds; `tools/redis-key-guard.sh` clean.
