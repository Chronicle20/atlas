# Monster Movement Local State (PS-3) — Design

Version: v1
Status: Approved for planning
Created: 2026-07-02
PRD: `docs/tasks/task-120-monster-move-local-state/prd.md`

---

## 1. Summary

`movement.Processor.ForMonster` (`services/atlas-channel/atlas.com/channel/movement/processor.go:111`) pays one REST call to atlas-monsters on every monster move packet (`monster.GetById`, line 112) and a second REST call to atlas-data on basic-attack moves (`monsterinfo.GetById`, line 128). This design removes both from the steady-state path:

1. **Live-monster mirror** — a new in-process, tenant-scoped projection in `atlas-channel/monster`, keyed by monster object id, holding exactly the fields the movement path reads (field identity, template id, MP, max MP, controller-aggro flag). Seeded from the REST fetch the CREATED consumer handler already performs; updated by `monster_status_event` events; REST `GetById` demoted to a miss fallback that backfills the mirror.
2. **Template-info TTL cache** — an in-process, tenant-scoped positive/negative TTL cache fronting `atlas-channel/monster/information.GetById`, mirroring task-060's semantics (5 min positive / 30 s negative, env-configurable, `requests.ErrNotFound` classification) but memory-backed instead of Redis-backed, per the PRD's user decision.
3. **atlas-monsters MP-event closure** — the design phase confirmed three MP mutations that emit no event today (skill-cast deduct, basic-attack deduct, recovery regen). atlas-monsters gains additive `MP_CHANGED` emissions for all three, reusing the existing `mpChangedStatusEventProvider`, so mirror MP tracks registry MP.

No wire-format changes, no new topics, no new endpoints, no Redis. All three PRD open questions are resolved below (§3).

## 2. What the movement path actually reads (verified)

From `movement/processor.go:111-211`, the `mo` model obtained via REST is read for exactly:

| Read | Line | Use |
|---|---|---|
| `mo.WorldId() / ChannelId() / MapId()` | 118 | field-consistency rejection |
| `mo.Mp()` | 125 | `ackMp` forecast seed |
| `mo.MonsterId()` | 128, 130 | template-info lookup key (basic attacks only) |
| `mo.ControllerHasAggro()` | 144 | wire `useSkills` flag |

Everything else (`X/Y/Stance/Fh/Hp/...`) is unused here. The mirror entry carries this set plus `maxMp` (immutable per spawn, free at seed time, and required by the PS-1 MP Eater pre-screen in `monster.Processor.DrainMp` — keeps the mirror adoptable for PS-1 without redesign). HP is deliberately **not** tracked: `DAMAGED` events carry damage deltas, not absolute HP (`StatusEventDamagedBody`), so an HP projection would be drift-prone guesswork; PS-1 can extend the event surface when it needs HP.

## 3. Resolution of PRD open questions

### OQ1 — Initial MP at CREATED: seed from the CREATED handler's existing REST fetch

`handleStatusEventCreated` (`kafka/consumer/monster/consumer.go:120-155`) **already calls `monster.NewProcessor(l, ctx).GetById(e.UniqueId)`** (line 130) to build the spawn packet. The mirror is seeded from that full model — MP, max MP, aggro, field, and template id all present — at zero additional cost and zero wire change. This supersedes all three options listed in the PRD (template-sourced MP, CREATED-body enrichment, first-move fallback): no enrichment of the CREATED body is needed, and the first move after spawn is already mirror-warm. If that REST fetch fails, the handler's existing error path runs unchanged and the monster warms via the movement-path fallback instead.

### OQ2 — MP_CHANGED coverage: three confirmed gaps, closed additively in atlas-monsters

Verified MP mutation sites in atlas-monsters:

| Mutation | Site | Event today |
|---|---|---|
| MP Eater drain | `monster/processor.go` `DrainMp` (~1476) | `MP_CHANGED` (Reason `MP_EATER`) ✅ |
| Skill-cast `MpCon` deduct | `monster/processor.go:626-632` (`UseSkill`) | **none** ❌ |
| Basic-attack `ConMP` deduct | `monster/processor.go:823-827` (`UseBasicAttack`) | **none** ❌ |
| Recovery regen (`mpRecovery`, 10 s tick) | `recovery_task.go:113` — `ApplyRecovery` returns `(Model, hpApplied, mpApplied, error)` (`registry.go:497`) and the task **discards `mpApplied`** | **none** ❌ |

Without closure, mirror MP decays monotonically toward 0 for long-lived casters (regen invisible) and lags every cast — the ack would eventually tell the client the mob has 0 MP, and the client-side mob brain stops proposing `conMP` attacks. That is behavior drift versus today's REST read, so the PRD's conditional atlas-monsters scope is triggered.

Fix: emit `MP_CHANGED` at all three sites via the existing `mpChangedStatusEventProvider` (`monster/producer.go:124-139`), with new `Reason` constants `SKILL_CAST`, `BASIC_ATTACK`, `RECOVERY` (`CharacterId=0`; `SkillId` = mob skill id for `SKILL_CAST`, 0 otherwise; `MonsterMpAfter` from the post-deduct model already returned by `DeductMp`/`ApplyRecovery`). This is additive and mixed-version safe: atlas-channel's existing `handleStatusEventMpChanged` switch (`consumer.go:574-604`) routes unknown reasons to the `default:` debug-log branch and does nothing.

Event volume: `SKILL_CAST`/`BASIC_ATTACK` fire only on actual casts/attacks with MP cost. `RECOVERY` fires at most once per 10 s per monster that is below max MP **and** has `mpRecovery > 0` (gated on `mpApplied`), the same order of magnitude as the HP-bar heal events the task already emits. Alternatives rejected: channel-side regen modeling (duplicates the recovery formula, tick-alignment drift) and accepting staleness (the drift described above).

### OQ3 — Staleness sweep: coarse lastWrite TTL, constants not env

Mirror entries are evicted by `DESTROYED`/`KILLED` (same call sites where `StatusMirror.OnMonsterGone` already runs, `consumer.go:184, 268`). A leak requires the entry's death event to be missed while the pod stays up (consumer rebalance gap) — rare, and each leaked entry is ~100 bytes. Defensive sweep: a `time.Ticker` goroutine started with the mirror singleton scans every **5 minutes** and evicts entries whose `lastWrite` is older than **30 minutes**. Eviction of a still-live entry is harmless: the next move takes one REST fallback and re-backfills. Constants (not env vars) — this is leak insurance, not a tuning surface. Touch-on-read was rejected: it would put a write (or per-entry atomic) on the hot read path for marginal benefit.

## 4. Alternatives considered

**Live state.**
- **A (chosen): event-projected in-process mirror + REST miss-fallback.** Follows the proven `StatusMirror`/`NextSkillInbox` pattern in the same package; correctness never depends on projection freshness because the fallback is authoritative.
- **B: Redis-backed shared cache.** Rejected — explicit PRD non-goal (user decision); still a network hop per move; PS-4 documents Redis costs as its own problem.
- **C: short-TTL memoization of `monster.GetById`.** Simpler, but any TTL long enough to help serves stale aggro/MP with no invalidation signal, and it ignores the event stream the service already consumes.
- **D: move ack computation into atlas-monsters.** Ownership inversion, adds a Kafka round-trip to a packet ack, changes wire timing. Rejected.

**Template info.**
- **A (chosen): channel-local in-process TTL cache** with task-060's TTL/negative/classification semantics. task-060 explicitly deferred this exact follow-up to a channel-local implementation.
- **B: reuse task-060's Redis cache namespace.** Rejected — PRD non-goal (no Redis hop), and cross-service cache coupling.
- **C: extract a shared TTL-cache lib.** Two implementations exist (Redis-backed in atlas-monsters, in-process here) with different backends and ~100 lines each; a lib abstraction is YAGNI now. Revisit if a third consumer appears.

## 5. Component design

### 5.1 `monster.LiveMirror` (new: `services/atlas-channel/atlas.com/channel/monster/live_mirror.go`)

```
type LiveEntry struct {
    Field              field.Model   // world/channel/map/instance
    MonsterId          uint32        // template id
    Mp                 uint32
    MaxMp              uint32
    ControllerHasAggro bool
    LastWrite          time.Time
}

type LiveMirror struct {
    mu        sync.RWMutex
    perTenant map[uuid.UUID]map[uint32]LiveEntry   // tenant -> uniqueId -> entry
}
```

Singleton via `sync.Once` (`GetLiveMirror()`), per the `StatusMirror` precedent. API:

- `Lookup(t tenant.Model, uniqueId uint32) (LiveEntry, bool)` — RLock read.
- `Put(t, uniqueId, entry)` — full-entry write. Used by the CREATED seed and the movement fallback backfill.
- `UpdateMp(t, uniqueId, mpAfter uint32)` / `UpdateAggro(t, uniqueId, aggro bool)` — **update-only**: no-op when the entry is absent. Events must never create entries, because the event envelope cannot supply `ControllerHasAggro`/`MaxMp`; a partial entry with a defaulted-false aggro flag would make the client render the mob idle (wire `useSkills=false`). Absent entry ⇒ the movement fallback creates an authoritative one.
- `Remove(t, uniqueId)` — DESTROYED/KILLED eviction.
- `EvictTenant(tid uuid.UUID)` — wired into the existing `listener.RegisterEvictor` block (`main.go:287`).
- `SweepStale(now, maxAge)` — used by the sweep ticker (§3 OQ3); also invoked directly by tests.

Every write refreshes `LastWrite`.

### 5.2 Mirror write paths (all inline additions to existing consumer handlers, `StatusMirror` precedent)

| Event | Handler | Mirror action |
|---|---|---|
| `CREATED` | `handleStatusEventCreated` | `Put` full entry from the already-fetched model `m` (after the `sc.Is` gate, before/independent of packet emission) |
| `DESTROYED` / `KILLED` | existing handlers | `Remove` (next to the existing `StatusMirror.OnMonsterGone` calls) |
| `START_CONTROL` | `handleStatusEventStartControl` | `UpdateAggro(e.Body.ControllerHasAggro)` |
| `STOP_CONTROL` | `handleStatusEventStopControl` | `UpdateAggro(false)` — no controller ⇒ no aggro |
| `AGGRO_CHANGED` | `handleStatusEventAggroChanged` | `UpdateAggro(e.Body.ControllerHasAggro)` |
| `MP_CHANGED` (any Reason) | `handleStatusEventMpChanged` | `UpdateMp(e.Body.MonsterMpAfter)` — placed immediately after the type/tenant gates and **before** the session lookup early-return at `consumer.go:569-571` and before the Reason switch, so new Reasons and absent sessions still update the mirror |

Existing packet-emitting behavior in every handler is untouched (PRD FR-1.3). Updates use absolute values (`MonsterMpAfter`, aggro bool), never deltas — last-writer-wins self-corrects any narrow reorder window. Kafka ordering per monster is preserved by the topic's keying; the known seed race (an event landing between the CREATED handler's REST read and its `Put`) has a millisecond window at spawn time when MP is at max, and self-corrects on the next MP event. Accepted; documented in code comment.

### 5.3 Movement path consumption (`movement/processor.go`)

`ForMonster` replaces the unconditional REST call with:

```
entry, ok := monster.GetLiveMirror().Lookup(p.t, objectId)
if !ok {
    mo, err := monsterByIdFn(p.l, p.ctx, objectId)   // package-level seam, default = monster.NewProcessor(l, ctx).GetById
    if err != nil { /* identical log + return err as today (lines 113-116) */ }
    entry = liveEntryFrom(mo)
    monster.GetLiveMirror().Put(p.t, objectId, entry)
    // debug log + fallback metric (FR-4.2)
}
```

All downstream reads switch to `entry` fields: field-consistency check (`entry.Field` vs `f`, same rejection), `ackMp := uint16(entry.Mp)`, `monsterinfo` lookup key `entry.MonsterId`, `useSkills := entry.ControllerHasAggro`. The inbox take-and-clear, packet writers, snap logic, and Kafka command emission are untouched — ack/broadcast bytes and commands are identical for identical logical state (PRD acceptance).

`monsterByIdFn` is a package-level var seam (precedent: `monsterStatSetBroadcaster` spy vars in `consumer.go:362`, `upstreamFn` in task-060's cache) so tests can prove "zero REST on warm path" by injecting a failing/counting fake.

### 5.4 Template-info TTL cache (`services/atlas-channel/atlas.com/channel/monster/information/cache.go`)

Transparent behind the existing `Processor.GetById` signature (FR-3.2). Structure:

```
type cacheEntry struct { model Model; negative bool; expiresAt time.Time }
perTenant map[uuid.UUID]map[uint32]cacheEntry   // tenant -> templateId -> entry
```

Singleton + `sync.RWMutex`; lazy expiry on read (expired ⇒ miss ⇒ refetch-and-overwrite); no sweep needed (population is O(distinct templates), entries are overwritten in place). `EvictTenant` wired into the same evictor block.

Semantics ported from task-060's `cache.go` (env parsing, classification, error synthesis — same shapes, memory-backed):

- Env: `MONSTER_INFO_CACHE_ENABLED` (default true), `MONSTER_INFO_CACHE_TTL` (default 5 m, clamp [1 s, 24 h]), `MONSTER_INFO_CACHE_NEGATIVE_TTL` (default 30 s, clamp [0 s, 5 m]). Read once via `sync.Once`; invalid values warn and fall back to defaults.
- Negative caching only for `errors.Is(err, requests.ErrNotFound)`; transient errors (network/5xx/parse) are never cached. Negative hits synthesize an error wrapping `requests.ErrNotFound` so callers see the same shape as a live 404.
- Concurrent same-key misses may duplicate the upstream fetch (no singleflight) — bounded by template count, matches task-060's accepted behavior.
- The existing `upstreamFn`-style indirection is reused for tests.

### 5.5 Observability

atlas-channel gains its first Prometheus wiring, copying the task-060 pattern: `promauto` counters in the owning packages plus `AddRouteInitializer(server.MountHandler("/metrics", promhttp.Handler()))` in `main.go` (precedent: `services/atlas-monsters/atlas.com/monsters/main.go:96`). Note: `MountHandler` mounts under the REST base path, so the endpoint is **`/api/metrics`** — scrape config must use that path (known readiness-probe analogue).

Counters (all labeled `tenant`):

- `atlas_channel_monster_mirror_hits_total`
- `atlas_channel_monster_mirror_misses_total`
- `atlas_channel_monster_mirror_fallback_total{outcome="success|failure"}`
- `atlas_channel_monster_info_cache_hits_total{kind="positive|negative"}`
- `atlas_channel_monster_info_cache_misses_total`

Mirror misses on the movement path additionally log at debug with the object id (FR-4.2).

### 5.6 atlas-monsters changes (additive only)

- `kafka.go`: add `MpChangeReasonSkillCast = "SKILL_CAST"`, `MpChangeReasonBasicAttack = "BASIC_ATTACK"`, `MpChangeReasonRecovery = "RECOVERY"`.
- `UseSkill`: after the successful `DeductMp` (`processor.go:628`), emit `MP_CHANGED` from the returned post-deduct model (capture the currently-discarded return).
- `UseBasicAttack`: same after `processor.go:823`.
- `recovery_task.go`: capture `mpApplied` (line 113) and, when true, emit `MP_CHANGED` (Reason `RECOVERY`) via the injectable `emitFn` seam family (add an `mpEmitFn` or widen `emitFn` — implementation detail; tests already inject these seams).
- Mirror of these constants in atlas-channel's `kafka/message/monster/kafka.go` for the consumer side (constants only; no handler behavior change).

Deploy-order free: old channel + new monsters ⇒ unknown Reasons hit the debug default; new channel + old monsters ⇒ mirror MP lags exactly as far as today's post-command REST read could, and self-corrects via fallback/sweep.

## 6. Failure modes

- **REST fallback fails**: identical log + error return as today (`processor.go:113-116`). PRD FR-2.2 preserved.
- **Kafka consumer lag/outage**: mirror serves last-known state; field identity is fixed per spawn and template id immutable, so the only drift surface is MP/aggro — the same drift today's flow has between a Kafka command being emitted and atlas-monsters processing it. Dead monsters linger until the sweep, and a move for a dead monster takes the fallback path, reproducing today's error behavior (PRD acceptance).
- **Pod restart**: `kafka.LastOffset` ⇒ empty mirror ⇒ warm via CREATED seeds (new spawns) and movement fallbacks (existing mobs). First move per existing monster costs one REST — exactly today's cost, never worse.
- **CREATED-handler REST failure**: no seed; movement fallback covers.

## 7. Testing

Builder-pattern setup throughout; no `*_testhelpers.go`.

- **LiveMirror unit**: seed/lookup round-trip; `UpdateMp`/`UpdateAggro` no-op on absent entries; DESTROYED/KILLED removal; STOP_CONTROL aggro-false; tenant eviction; `SweepStale` boundary cases; cross-tenant isolation (PRD acceptance); concurrent read/write under `-race`.
- **Movement processor**: warm-path zero-REST (counting fake in `monsterByIdFn`); miss ⇒ exactly one fallback ⇒ backfill ⇒ second move REST-free; fallback error preserved; field-mismatch rejection unchanged; `ackMp`/`useSkills` equivalence against a table of mirror states (regression per PRD acceptance).
- **Template cache**: positive hit/expiry; negative hit with shorter TTL; transient errors uncached; env parsing (invalid ⇒ default); disabled flag passthrough; tenant isolation; `-race`.
- **Consumer handlers**: each event type mutates the mirror as §5.2, and MP updates land before the session-lookup early return and for unknown Reasons.
- **atlas-monsters**: `MP_CHANGED` emitted on skill deduct, basic-attack deduct, and `mpApplied` recovery (existing injectable seams); no emission when deduct/regen doesn't happen.
- **Verification gate** (CLAUDE.md): `go test -race ./...`, `go vet ./...`, `go build ./...` in atlas-channel and atlas-monsters; `docker buildx bake atlas-channel atlas-monsters`; `tools/redis-key-guard.sh`.

## 8. Out of scope (unchanged from PRD)

PS-1 (attack-path fan-out — mirror is shaped for it, adoption is a later task), PS-2 (`ForOtherSessionsInMap` broadcast REST), PS-4 (Redis locking in atlas-monsters), atlas-data changes, Redis-backed caching, event-driven template-cache invalidation, other `monster.GetById` call sites in atlas-channel.
