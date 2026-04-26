# Monster Aggro & Controller Switching — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-26
---

## 1. Overview

Atlas currently assigns a monster controller exactly once — when the monster spawns — and only revisits the assignment when the controller disconnects, leaves the map, or the monster dies. The controlling client is responsible for driving monster movement and skill cadence, so a back-row mage tagging a mob first ends up controlling movement for a warrior who's actually tanking. Movement stutters as the controller's distance changes; mob-cast skills fire at the wrong cadence; and bosses behave indistinguishably from field mobs once the initial controller assignment is made.

This feature adds a per-monster damage-attribution table keyed by attacker character ID, with each entry tracking accumulated damage and the wall-clock time of the last hit. The damage table drives three behaviors: (1) the controller is reassigned to whoever currently leads damage-per-second when a non-controlling player lands a hit, (2) a `controllerHasAggro` flag is broadcast with each control event so the controller's client knows whether to render the monster as "active" (aggressive, auto-attack timer running) or "passive" (idle), and (3) a background sweep decays damage entries on Cosmic's exponential backoff schedule and, when all entries expire on a non-boss, clears the controller so the monster can re-settle.

Monsters flagged as bosses (`information.Boss()`) are exempt from decay-driven controller clearing — they retain their controller and aggro state until death. This work is a prerequisite for Spec-Task 2 (mob-skill picker), which will read the controller and aggro state to decide whether to fire active-only skills.

## 2. Goals

Primary goals:
- Reassign monster controllers to the current DPS leader on damage, so movement and skill cadence track the player who's actually fighting.
- Broadcast a two-state `controllerHasAggro` flag end-to-end (atlas-monsters → atlas-channel → client) so the controller's client distinguishes active vs. passive control.
- Decay damage entries on a background sweep so non-boss monsters fall out of combat naturally when no one is hitting them.
- Treat boss monsters as combat-locked: their controller and aggro state persist until death.
- Establish the damage-table schema and event contract that Spec-Task 2 will consume for mob-skill selection.

Non-goals:
- Out-of-combat HP regeneration. Cosmic does not reset HP on aggro clear and v83 has no in-combat regen; this PRD explicitly skips both.
- Puppet aggro redirect (Rogue/Cleric summon mechanic). Depends on summon-actor wiring not yet present.
- Cross-channel aggro persistence. Monsters are channel-scoped; aggro tables follow the monster.
- Tenant-configurable decay parameters. Cosmic's constants are hardcoded for this task; lifting to per-tenant config is a follow-up if a tenant ever needs to tune them.
- Status-effect interactions with aggro (stuns flipping the leader, taunt, etc.). Those are separate features.

## 3. User Stories

- As a warrior tanking a mob, I want the mob's movement controller to be on me (not the mage who tagged it from across the map) so the AI moves toward me at a sensible cadence.
- As a mage who tags a mob and walks away, I want to lose movement control to whoever is actually fighting it so I'm not paying client-side AI cost for a fight I'm not in.
- As any party member, I want a mob I'm not currently fighting to settle back to its idle/passive state after a few seconds so it doesn't keep tracking me indefinitely.
- As a boss raider, I want the boss to remain locked-on to its current controller and aggro state for the entire fight regardless of damage gaps.
- As an operator, I want to see in monitoring/logs when a controller switch or aggro decay happens so I can debug movement complaints.

## 4. Functional Requirements

### 4.1 Damage-attribution table

- **FR-1.** Each monster's damage table is keyed by `characterId`. Entries are aggregated at write time: one row per attacker holding `(characterId, totalDamage, lastHitMs)`. Append-per-line behavior is removed.
- **FR-2.** `lastHitMs` is recorded on every successful damage application (including DoT ticks via `DamageSourceDamageOverTime`) using the wall-clock time of the write. Time source: server `time.Now().UnixMilli()` evaluated in Go at call time and passed into the Lua script — *not* `redis.call('TIME', ...)`, to keep behavior matchable in unit tests.
- **FR-3.** Reflect damage (`checkReflect` path) does not write damage entries. The character takes damage; the monster's aggro table is unaffected. (Matches today's behavior; explicit so the implementer doesn't add it.)
- **FR-4.** `DamageSourceHeal` events do not write damage entries. (Healing should not give the healer aggro on the monster.)
- **FR-5.** When a monster is destroyed (death, despawn, or map clear), its damage table is discarded along with the rest of the monster state — no separate cleanup hook needed.

### 4.2 Controller switching on damage

- **FR-6.** When `Damage` is called with a non-zero damage line and the attacker is *not* the current controller, atlas-monsters checks whether the attacker now leads the DPS list (highest `totalDamage` among entries within the active window — see §4.4 for window semantics). If yes, switch controller.
- **FR-7.** Controller switch emits two events on `EVENT_TOPIC_MONSTER_STATUS`, in this order, both partition-keyed by `uniqueId`:
  1. `STOP_CONTROL` with the *previous* controller's character ID in `actorId`.
  2. `START_CONTROL` with the *new* controller's character ID in `actorId`, plus `controllerHasAggro: true`.
- **FR-8.** Damage that does not result in a controller switch must NOT emit `STOP_CONTROL` / `START_CONTROL`. (Don't churn the channel for every hit.)
- **FR-9.** If the current controller is `0` (no controller — possible after a decay clear), the next attacker becomes the controller via `START_CONTROL` with no preceding `STOP_CONTROL`.
- **FR-10.** Controller switching is gated to the same field as the current controller. If a damage event arrives from a character not in the monster's field, ignore it for controller selection (still apply damage).

### 4.3 `controllerHasAggro` flag

- **FR-11.** A new boolean field `controllerHasAggro` is added to:
  - `storedMonster` (Redis representation) and `Model` (in-memory).
  - `statusEventStartControlBody` (Kafka event body).
  - A new `statusEventAggroChangedBody` (see §4.5).
- **FR-12.** Initial controller assignment on monster creation sets `controllerHasAggro: false` (passive — the monster is in idle/wander mode, not actively aggressive).
- **FR-13.** When the first damage entry is written for any attacker AND a controller exists, `controllerHasAggro` flips to `true`. If this flip happens without a controller change, emit a dedicated `AGGRO_CHANGED` event (§4.5).
- **FR-14.** Controller switches (FR-7) always emit `START_CONTROL` with `controllerHasAggro: true` because by definition the switch was triggered by damage.
- **FR-15.** When the decay sweep clears all damage entries on a non-boss (§4.4), `controllerHasAggro` flips to `false` and the controller is cleared (FR-19). No `AGGRO_CHANGED` event is needed in that case because the `STOP_CONTROL` event communicates the state change.

### 4.4 Aggro decay sweep

- **FR-16.** A new background task (`MonsterAggroDecayTask`) runs on the existing `tasks` package with a 1500ms interval, mirroring Cosmic's `MonsterAggroCoordinator` cadence.
- **FR-17.** On each tick, the task iterates all monsters across all tenants (same scan pattern as `StatusExpirationTask`). For each monster:
  - Skip if `information.Boss()` returns true.
  - Skip if the damage table is empty.
  - For each damage entry, if `now - lastHitMs > AggroIdleThresholdMs`, decay the entry by multiplying `totalDamage` by `AggroDecayMultiplier`. If the decayed value falls below `AggroDecayFloor`, remove the entry.
  - If all entries are removed and the monster is non-boss: clear the controller via the existing `StopControl` path (emits `STOP_CONTROL`), set `controllerHasAggro: false`.
- **FR-18.** Constants live in `monster/aggro.go` as named exported values:
  - `AggroIdleThresholdMs` — duration before decay starts on an entry. Default: Cosmic's value (10000ms / 10s).
  - `AggroDecayMultiplier` — multiplicative decay per tick once idle. Default: Cosmic's value (0.85 — i.e., 15% reduction per 1.5s tick once idle).
  - `AggroDecayFloor` — minimum damage value before entry is removed. Default: Cosmic's value (1).
  - `AggroSweepIntervalMs` — sweep cadence. Default: 1500ms.
  
  Implementer must verify exact Cosmic values from `MonsterAggroCoordinator.java:110-148` and document in code comments which Cosmic constants each maps to.
- **FR-19.** Clearing the controller via the decay path emits `STOP_CONTROL` with the previous controller's character ID. The next observer-controlled-monster tick (existing logic in `processor.go`) reassigns a controller if the monster is still in someone's field.
- **FR-20.** Decay sweep does NOT emit any per-entry "decayed" events — this is internal state.

### 4.5 New `AGGRO_CHANGED` event

- **FR-21.** New event type `AGGRO_CHANGED` on `EVENT_TOPIC_MONSTER_STATUS`. Body:
  ```go
  type statusEventAggroChangedBody struct {
      ControllerCharacterId uint32 `json:"controllerCharacterId"`
      ControllerHasAggro    bool   `json:"controllerHasAggro"`
  }
  ```
- **FR-22.** Emitted only when `controllerHasAggro` changes value AND there is no accompanying `STOP_CONTROL` or `START_CONTROL` event. (i.e., the flag flipped without the controller changing.)
- **FR-23.** atlas-channel consumes this event and re-sends `MonsterControlWriter` with the appropriate `ControlType` (active vs. passive) to the controller's session — same packet shape as `START_CONTROL`, just without changing who's controlling.

### 4.6 Boss exemption

- **FR-24.** Boss flag is read from `information.GetById(p.l)(p.ctx)(m.MonsterId()).Boss()`. The decay task caches the boss flag per monster-template-id within a single tick to avoid redundant lookups.
- **FR-25.** Bosses do NOT participate in the decay sweep at all — their damage entries persist until death. (Controller switching on damage still applies to bosses. A boss can change controllers if a different player takes the DPS lead.)

## 5. API Surface

### 5.1 Kafka events (atlas-monsters → atlas-channel)

Topic: `EVENT_TOPIC_MONSTER_STATUS` (existing). Partition key: `uniqueId` (existing).

**Modified event:** `START_CONTROL`
```json
{
  "uniqueId": 5001,
  "monsterId": 100100,
  "type": "START_CONTROL",
  "body": {
    "actorId": 4242,
    "x": 100,
    "y": 200,
    "stance": 5,
    "fh": 12,
    "team": 0,
    "controllerHasAggro": true
  }
}
```
- Adds `controllerHasAggro` to the body. atlas-channel uses it when calling `StartControlMonsterBody(m, controllerHasAggro)`.

**New event:** `AGGRO_CHANGED`
```json
{
  "uniqueId": 5001,
  "monsterId": 100100,
  "type": "AGGRO_CHANGED",
  "body": {
    "controllerCharacterId": 4242,
    "controllerHasAggro": false
  }
}
```

`STOP_CONTROL`, `DAMAGED`, `KILLED`, `CREATED`, `DESTROYED`, status events: unchanged.

### 5.2 Channel-side packet handling

No new packets — `MonsterControlWriter` already supports active/passive control types via `ControlTypeActiveRequest` (aggro) vs `ControlTypeActiveInit` (passive). The channel's `handleStatusEventStartControl` is updated to read `e.Body.ControllerHasAggro` and pass it to `StartControlMonsterBody` (which today receives a hardcoded `false` at `consumer.go:241`).

A new handler `handleStatusEventAggroChanged` is added to atlas-channel to respond to `AGGRO_CHANGED`. It looks up the monster, locates the session for `ControllerCharacterId`, and re-sends `MonsterControlWriter` with the new aggro state. No `STOP_CONTROL` is emitted to the client.

### 5.3 REST surface

No REST changes. The damage table is internal to atlas-monsters and not exposed via the existing monster REST endpoints. (Spec-Task 2 may expose a read-only DPS-leader query if needed; that's deferred.)

## 6. Data Model

### 6.1 Damage entry — Redis representation

**Before (current):**
```go
type storedDamageEntry struct {
    CharacterId uint32 `json:"characterId"`
    Damage      uint32 `json:"damage"`
}
```
Multiple rows per character (one per damage line).

**After:**
```go
type storedDamageEntry struct {
    CharacterId uint32 `json:"characterId"`
    Damage      uint32 `json:"damage"`     // aggregated total
    LastHitMs   int64  `json:"lastHitMs"`  // wall-clock unix milliseconds
}
```
One row per character. `Damage` accumulates across all hits.

### 6.2 Monster model field additions

```go
type Model struct {
    // ... existing fields ...
    controllerHasAggro bool
}

type storedMonster struct {
    // ... existing fields ...
    ControllerHasAggro bool `json:"controllerHasAggro"`
}
```

Builder gets `SetControllerHasAggro(bool)`. Model getter `ControllerHasAggro() bool`.

### 6.3 Lua script change

`applyDamageScript` is rewritten to:
1. Decode current `damageEntries`.
2. Find existing entry by `characterId` — if found, increment `damage` and overwrite `lastHitMs`. If not, append new entry.
3. Set `controllerHasAggro = true` if a controller exists and the flag is currently false (the Go caller decides whether to emit `AGGRO_CHANGED` based on the before/after state — Lua returns both).
4. Re-encode and SET.

Script returns the updated `storedMonster` JSON (same as today) so the Go caller can decode the post-state.

### 6.4 Migration / in-flight monsters

Monsters spawned before the upgrade have:
- Multiple `damageEntries` rows per character (legacy append-per-line).
- No `lastHitMs` field.
- No `controllerHasAggro` field.

Strategy: **eat the legacy state on first read.**
- `fromStored` collapses multiple entries by `characterId` into a single entry, summing `damage` and treating missing `lastHitMs` as `0` (which means "instantly stale" — they'll be decayed on the first sweep).
- Missing `controllerHasAggro` defaults to `false`.

This is acceptable because monster state is per-channel and bounded — within a few minutes of deploy all monsters will have been respawned or aggro-cleared.

## 7. Service Impact

### 7.1 atlas-monsters

**New files**
- `monster/aggro.go` — exported decay constants + helper `IsAggroIdle(entry storedDamageEntry, now int64) bool`.
- `monster/aggro_task.go` — `MonsterAggroDecayTask` implementing the `tasks.Task` interface.

**Modified files**
- `monster/registry.go` — `applyDamageScript` rewritten; `storedDamageEntry` gets `LastHitMs`; `storedMonster` gets `ControllerHasAggro`; `fromStored` migrates legacy state; `toStored` writes new fields. New atomic update `SetControllerAndAggro(t, uniqueId, ctrlId, hasAggro)` for controller-switch path. New atomic update `DecayDamageEntries(t, uniqueId, now)` returning the entries that were removed (so the task can decide whether to clear the controller).
- `monster/model.go` — `controllerHasAggro` field; getter; `Model.Damage` updated to set the new field semantics; legacy `entry` struct gets `LastHitMs`. `DamageEntries()` and `DamageSummary()` collapse to identical behavior (one is removed or the other becomes a passthrough).
- `monster/builder.go` — `SetControllerHasAggro(bool)`.
- `monster/processor.go` — `Damage` method: after `ApplyDamage` succeeds, evaluate DPS leadership. If leader changed and is not current controller, call `StopControl` then `StartControl` (new helper `SwitchControl(uniqueId, newControllerId)` to keep this atomic from the Lua perspective). After damage, evaluate `controllerHasAggro` flip. New method `EmitAggroChanged(m, controllerId, hasAggro)` for the standalone flag flip.
- `monster/producer.go` — `startControlStatusEventProvider` adds `controllerHasAggro` to body; new `aggroChangedStatusEventProvider`.
- `monster/kafka.go` — new event constant `EventMonsterStatusAggroChanged = "AGGRO_CHANGED"`; new `statusEventAggroChangedBody` type; `statusEventStartControlBody` adds field.
- `main.go` (or wherever tasks register) — register `MonsterAggroDecayTask` alongside `StatusExpirationTask`.

**Tests**
- `monster/registry_test.go` — update for aggregated damage entries; new tests for decay script and migration of legacy state.
- `monster/processor_test.go` — controller-switch-on-DPS-lead, controller-no-switch-when-already-leader, aggro flag flip, no-event-when-flag-stable.
- `monster/aggro_task_test.go` (new) — decay over multiple ticks, full-clear path emits STOP_CONTROL on non-boss, boss is exempt.

### 7.2 atlas-channel

**Modified files**
- `kafka/message/monster/kafka.go` — add `ControllerHasAggro bool` to `StatusEventStartControlBody`; add `EventStatusAggroChanged` constant; add `StatusEventAggroChangedBody`.
- `kafka/consumer/monster/consumer.go` — `handleStatusEventStartControl` reads `e.Body.ControllerHasAggro` and passes to `StartControlMonsterBody` (replacing today's hardcoded `false` at `consumer.go:241`). New `handleStatusEventAggroChanged` registered as a persistent handler.

**Tests**
- New consumer test for `AGGRO_CHANGED` flow.
- Update existing `START_CONTROL` test fixtures with the new field.

### 7.3 libs/atlas-packet

**No changes.** `clientbound/control.go` already supports the active/passive control types. The aggro flag is a transport-level concern between atlas-monsters and atlas-channel; the wire packet is unchanged.

### 7.4 Documentation

- `services/atlas-monsters/docs/kafka.md` — document new `AGGRO_CHANGED` event and the `controllerHasAggro` field on `START_CONTROL`. Document the decay sweep behavior under "Background Tasks."
- `services/atlas-channel/docs/kafka.md` — update `START_CONTROL` consumer description and add `AGGRO_CHANGED` consumer entry.

## 8. Non-Functional Requirements

### 8.1 Performance

- Decay sweep must remain bounded. Current `StatusExpirationTask.Run()` iterates all monsters across all tenants every tick — same pattern. With ~10k active monsters and a 1500ms tick, the sweep needs to stay under ~50ms wall time to avoid drift. The decay check is in-process (read damage entries, check timestamps, compute multiplier) for monsters that are *idle*; only when an entry decays does it hit Redis via an atomic update. This should keep per-tick Redis traffic low.
- Damage-on-attack DPS-leader check is O(N) over the damage entries (N = distinct attackers, typically ≤6 for a party). Negligible.

### 8.2 Multi-tenancy

- Damage tables are per-monster, monsters are per-tenant; tenancy falls out for free via the existing `monsterKey(t, uniqueId)`.
- Decay task uses the same `tenant.WithContext` pattern as `StatusExpirationTask` when emitting events.
- All Kafka events carry the standard tenant headers via the existing `producer.ProviderImpl` path.

### 8.3 Observability

- Log at debug level on every controller switch with old/new IDs and the DPS leader's accumulated damage at switch time.
- Log at debug level on each decay-driven controller clear with the monster's `uniqueId` and the count of entries that aged out.
- Existing damaged-event logging remains.
- No new metrics required for v1; if controller-churn becomes a concern post-deploy we can add a counter.

### 8.4 Concurrency

- The Lua script keeps damage application and aggro-flag updates atomic per monster.
- Controller switching (`SwitchControl`) is a two-step Redis operation today (`ClearControl` then `ControlMonster` — see `processor.go:210-216`). Two concurrent damage events for the same monster could interleave, producing redundant `STOP_CONTROL`/`START_CONTROL` pairs. This is acceptable: the events are partition-ordered by `uniqueId` and the channel is idempotent for re-control to the same character. Keeping today's two-call structure rather than introducing a new combined Lua script. (Implementer should add a comment noting this design choice.)

### 8.5 Failure modes

- If atlas-monsters crashes mid-sweep, in-flight Redis state is consistent (each `DecayDamageEntries` call is atomic via Lua). On restart the sweep resumes from current state.
- If atlas-channel is offline when `AGGRO_CHANGED` is emitted, the monster's client-side aggro rendering is briefly stale; once the channel reconnects, the next damage or controller change re-syncs. This is acceptable for v1.

## 9. Open Questions

- **OQ-1.** Exact Cosmic constants in `MonsterAggroCoordinator.java:110-148`. PRD assumes `idleThreshold=10000ms`, `decayMultiplier=0.85`, `decayFloor=1` based on the brief, but the implementer must verify by reading the source. If Cosmic's actual numbers diverge, follow Cosmic.
- **OQ-2.** Spec-Task 2 (mob-skill picker) will need a way to read the current DPS leader and aggro state from atlas-monsters. PRD does not pre-design that read API; it can be added when Spec-Task 2 lands. Implementer should keep `Model.ControllerHasAggro()` and DPS-leader logic exported / reachable so the future skill-picker can hook in without refactoring.

## 10. Acceptance Criteria

- [ ] Damage entries in Redis are aggregated per attacker with `lastHitMs` populated; legacy multi-row entries are migrated on first read without errors.
- [ ] When a non-controller attacker takes the DPS lead via `Damage`, atlas-monsters emits `STOP_CONTROL` (old controller) followed by `START_CONTROL` (new controller, `controllerHasAggro: true`) on the same partition.
- [ ] When the current controller takes additional damage and is still the DPS leader, no `STOP_CONTROL` / `START_CONTROL` is emitted.
- [ ] First damage on a freshly-spawned monster flips `controllerHasAggro` from `false` to `true` and emits exactly one `AGGRO_CHANGED` event (no `START_CONTROL` because controller didn't change).
- [ ] `MonsterAggroDecayTask` runs every 1500ms, decays idle entries on Cosmic's exponential schedule, and removes entries that fall below the floor.
- [ ] When a non-boss monster's damage table fully clears via decay, it emits `STOP_CONTROL` with the previous controller's ID and `controllerHasAggro` resets to `false`. Boss monsters never trigger this path.
- [ ] atlas-channel's `handleStatusEventStartControl` passes `controllerHasAggro` through to `StartControlMonsterBody` instead of hardcoded `false`. Controller's client renders the active vs. passive control type accordingly.
- [ ] atlas-channel's new `handleStatusEventAggroChanged` re-sends `MonsterControlWriter` to the controller's session with the updated control type, without emitting `STOP_CONTROL`.
- [ ] Boss monster (`information.Boss() == true`) retains its damage table and controller until death; aggro decay task skips it.
- [ ] Reflect damage and `DamageSourceHeal` events do NOT write damage entries.
- [ ] Existing tests in `monster/processor_test.go` and `monster/registry_test.go` are updated to reflect aggregated damage entries; new tests cover controller-switch, aggro flag flip, decay sweep, boss exemption, and legacy-state migration.
- [ ] `services/atlas-monsters/docs/kafka.md` and `services/atlas-channel/docs/kafka.md` are updated with the new event and modified body.
- [ ] Docker builds succeed for atlas-monsters and atlas-channel; full test suites pass for both services.
