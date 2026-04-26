# Monster Aggro & Controller Switching — Design

Version: v1
Status: Draft
Created: 2026-04-26
Companion to: [`prd.md`](./prd.md), [`data-model.md`](./data-model.md)

---

## 1. Purpose of this document

The PRD specifies *what* this feature does: per-monster damage tables, controller reassignment on DPS lead, two-state `controllerHasAggro` broadcast, decay sweep, and boss exemption. The data-model companion specifies the schema deltas. This design document records *how* — the architectural decisions left to the implementer, and the file-by-file shape of the change.

Five decisions were resolved during design:

| # | Decision | Choice |
|---|---|---|
| 1 | How does Go learn `controllerHasAggro` flipped during the damage script? | Lua returns `{wasFirstHit, monster}` envelope |
| 2 | How does the decay sweep run per-monster? | Hybrid: Go pre-filter on snapshot, Lua decay only when work is needed |
| 3 | How is FR-10 field-gating enforced? | Synchronous `CharacterIdsInFieldProvider` call, gated by existing precondition |
| 4 | Should controller switch be made atomic in one Lua? | No — keep two-step `StopControl` + `StartControl` per PRD §8.4 |
| 5 | When is `AGGRO_CHANGED` emitted within the multi-line damage flow? | At end of attack, iff `firstHitObserved && !controllerSwitched` |

Each decision is expanded in §3 with the considered alternatives.

---

## 2. Module structure

### 2.1 atlas-monsters

#### New files

**`monster/aggro.go`** — exports the four decay constants and a small pure helper.

```go
package monster

const (
    AggroIdleThresholdMs = int64(10000)
    AggroDecayMultiplier = 0.85
    AggroDecayFloor      = uint32(1)
)

const AggroSweepInterval = 1500 * time.Millisecond

// IsAggroIdle reports whether the entry's last hit is older than the idle threshold.
func IsAggroIdle(e entry, nowMs int64) bool {
    return nowMs-e.LastHitMs > AggroIdleThresholdMs
}
```

Doc comments describe what each constant controls. No external source references in code.

**`monster/aggro_task.go`** — the decay sweep task.

```go
type MonsterAggroDecayTask struct {
    l        logrus.FieldLogger
    ctx      context.Context
    interval time.Duration
}

func NewMonsterAggroDecayTask(l, ctx, interval) *MonsterAggroDecayTask
func (t *MonsterAggroDecayTask) Run()
func (t *MonsterAggroDecayTask) SleepTime() time.Duration
```

`Run()` mirrors `StatusExpirationTask.Run()` shape: iterate `GetMonsters()` per tenant, build a per-tick boss-flag cache (`map[uint32]bool` keyed by template `MonsterId()`, scoped to the single `Run()` invocation), then for each non-boss monster apply the hybrid decay flow described in §3.2.

#### Modified files

| File | Change |
|---|---|
| `registry.go` | `storedDamageEntry.LastHitMs int64` added; `storedMonster.ControllerHasAggro bool` added; `applyDamageScript` rewritten to upsert by `characterId` and return the `{wasFirstHit, monster}` envelope; new `decayDamageEntriesScript` Lua; new `Registry.DecayDamageEntries` method; `fromStored` collapses legacy multi-row entries; `toStored` writes new fields. `ApplyDamage` signature changes to accept `nowMs int64`; `DamageSummary` Go struct gains `WasFirstHit bool`. |
| `model.go` | `entry.LastHitMs int64` added; `Model.controllerHasAggro bool` added with `ControllerHasAggro() bool` getter; `Model.Damage(charId, damage)` updated to populate `LastHitMs` (used by tests / non-Lua paths only); `DamageSummary()` becomes a passthrough returning `m.damageEntries` directly since entries are now pre-aggregated; `DamageLeader()` is unchanged in semantics but operates on aggregated entries (same result, fewer iterations). |
| `builder.go` | `SetControllerHasAggro(bool) *Builder`. |
| `processor.go` | `Damage` rewritten per Decision 5 (track `firstHitObserved` and `controllerSwitched`); FR-10 field-membership check inserted before the existing controller-switch branch; `StartControl` updated so FR-9 (controller currently 0) skips the redundant `StopControl` call; new private `emitAggroChanged(m, hasAggro)` helper. The two-step switch keeps its existing shape with a comment noting the PRD §8.4 design choice. |
| `producer.go` | `startControlStatusEventProvider` accepts `hasAggro bool`; new `aggroChangedStatusEventProvider`. |
| `kafka.go` | `EventMonsterStatusAggroChanged = "AGGRO_CHANGED"`; `statusEventStartControlBody.ControllerHasAggro bool`; new `statusEventAggroChangedBody`. |
| `main.go` | Register `MonsterAggroDecayTask` alongside `StatusExpirationTask` and `DropTimerTask`. |

### 2.2 atlas-channel

| File | Change |
|---|---|
| `kafka/message/monster/kafka.go` | `EventStatusAggroChanged = "AGGRO_CHANGED"`; `StatusEventStartControlBody.ControllerHasAggro bool`; new `StatusEventAggroChangedBody`. |
| `kafka/consumer/monster/consumer.go` | Replace hardcoded `false` at `consumer.go:241` with `e.Body.ControllerHasAggro`. New `handleStatusEventAggroChanged` registered as a persistent handler in `InitHandlers`: looks up the monster via `monster.NewProcessor(...).GetById(e.UniqueId)`, finds the session for `e.Body.ControllerCharacterId`, re-sends `MonsterControlWriter` with `StartControlMonsterBody(m, e.Body.ControllerHasAggro)`. |

### 2.3 libs/atlas-packet

No changes. The wire packet's active/passive distinction is already supported via `ControlTypeActiveRequest` / `ControlTypeActiveInit` and consumed by `writer.StartControlMonsterBody(m, aggro)`.

### 2.4 Documentation

Updated alongside code (per PRD §7.4):

- `services/atlas-monsters/docs/kafka.md` — new `AGGRO_CHANGED` event entry; `controllerHasAggro` field added to `START_CONTROL` body; new "Background Tasks" entry for `MonsterAggroDecayTask`.
- `services/atlas-channel/docs/kafka.md` — new consumer entry for `AGGRO_CHANGED`; `START_CONTROL` consumer description updated.

---

## 3. Architectural decisions

### 3.1 Lua return contract for first-hit detection (Decision 1)

**Problem.** FR-13 fires `AGGRO_CHANGED` when the first damage entry on a monster lands while a controller exists. Go needs to know whether a given `ApplyDamage` call caused that flip — but only the script knows authoritatively (the alternative, comparing pre/post state in Go, opens a TOCTOU window).

**Choice.** The script returns a JSON envelope:

```json
{ "wasFirstHit": true, "monster": { ... existing storedMonster ... } }
```

The Lua side knows the answer for free: it already inspects `m.controllerHasAggro` to decide whether to flip it.

```lua
local hadAggro = m.controllerHasAggro
local wasFirstHit = false
if m.controlCharacterId ~= 0 and not hadAggro then
    m.controllerHasAggro = true
    wasFirstHit = true
end
local result = {wasFirstHit = wasFirstHit, monster = m}
return cjson.encode(result)
```

Go decodes both fields and propagates `WasFirstHit` on `DamageSummary`.

**Alternatives rejected.**

- *Pre-fetch in Go and compare.* Requires GET + EVAL (two round trips), opens a race window where another goroutine could flip aggro between the two calls. Marginally simpler script, materially worse semantics.
- *Compare against the post-state Model only.* Doesn't work — by the time we have post-state, we've lost the pre-state needed to detect the *transition*.

### 3.2 Decay sweep — hybrid Go filter + conditional Lua write (Decision 2)

**Problem.** PRD §8.1 caps the sweep at ~50ms wall-time across ~10k monsters per tick. A naive "EVAL one Lua per monster every tick" would issue ~6.7k Redis EVALs/sec steady-state, most of which are no-ops (monster's not idle yet, or has no entries to decay).

**Choice.** Two-layer flow per monster:

1. **Pre-filter (Go, in-process).** `GetMonsters()` already loads every monster's full state into Go. The task iterates the in-memory copy and computes, for each non-boss monster: "does any entry need decay or pruning?" using the constants from `aggro.go`. If no, skip — no Redis write.
2. **Decay write (Lua, atomic).** When the pre-filter says "yes," the task invokes `decayDamageEntriesScript` against that monster's key. The Lua script re-reads the live state and re-applies the decay/prune logic itself (Go's pre-filter is just a "should we bother?" gate, not authoritative). The script returns the post-state, including a `controllerCleared bool` flag indicating whether all entries were pruned and the controller was cleared.
3. **Post-write emission (Go).** If `controllerCleared == true`, Go emits `STOP_CONTROL` for the previous controller, per FR-19. The next existing observer-controlled-monster reassignment tick (already in the codebase) will pick a new controller if anyone is still in-field.

**Race tolerance.** Between Go's pre-filter and the Lua write, another goroutine may land a hit, refreshing `lastHitMs`. The Lua's re-evaluation handles this correctly: on the post-hit state, the entry is no longer idle, so the script does nothing. The pre-filter false-positive is benign.

If the Go pre-filter misses a needed decay (because it observed stale state), the next tick (1.5s later) catches it. Decay is not time-critical to the millisecond.

**Alternatives rejected.**

- *Pure Lua per-monster every tick.* Simpler but needlessly chatty; PRD §8.1 explicitly endorses the hybrid shape.
- *All-Go decay via WATCH/MULTI/EXEC.* Loses atomicity-with-other-Lua-paths and forces an optimistic-lock retry loop on contention with `applyDamageScript`. Strictly worse.

### 3.3 FR-10 field-membership check (Decision 3)

**Problem.** FR-10 says: don't switch controller to an attacker who's no longer in the monster's field (e.g., a stale Kafka event from a character who warped out). The check has to fire at controller-switch time.

**Choice.** Use the existing `_map.CharacterIdsInFieldProvider(p.l)(p.ctx)(m.Field())()` synchronously inside `Damage`, gated by the existing precondition `characterId != m.ControlCharacterId() && m.DamageLeader() == characterId`.

The check fires only on a *would-be* controller switch — not per damage line, not per attack against the same controller. In practice that's a handful of calls per fight, not per hit. The cost is in line with the same provider's existing use in `getControllerCandidate` (`processor.go:128`) and `getDiseaseTargets` (`processor.go:717`).

```go
// inside Damage, after the per-line loop, before the existing switch branch:
if characterId != last.Monster.ControlCharacterId() && last.Monster.DamageLeader() == characterId {
    inField, err := p.attackerInField(last.Monster.Field(), characterId)
    if err != nil || !inField {
        // FR-10: skip switch but keep the damage we already applied
    } else {
        // existing StopControl + StartControl two-step
        controllerSwitched = true
    }
}
```

**Alternatives rejected.**

- *In-memory per-field membership cache populated by a new Kafka consumer in atlas-monsters.* Faster per-lookup but requires designing a new event-shape contract, restart-bootstrap, and consistency on edge cases (channel transfers, disconnects, instance teardown). Substantial new subsystem for a check that fires at sub-Hz frequency. Out of scope for v1.
- *Trust the channel; skip the check.* Tolerates a brief mis-control on stale events. Doesn't satisfy FR-10 as written.

### 3.4 Controller switch — keep two-step, no combined Lua (Decision 4)

**Problem.** Today's `StopControl(m)` then `StartControl(uniqueId, newId)` are two atomic Redis operations and emit two Kafka events. Two concurrent damage events for the same monster could interleave, producing redundant `STOP_CONTROL`/`START_CONTROL` pairs.

**Choice.** Keep the existing two-step structure. Add a code comment in `processor.go` next to the switch site explaining the design choice (per PRD §8.4):

> Controller switching uses two separate atomic ops by design. Two concurrent damage events for the same monster could interleave and produce redundant `STOP_CONTROL`/`START_CONTROL` pairs; this is acceptable because partition-keyed Kafka delivery preserves ordering and the channel re-applies idempotently for re-control to the same character.

**Alternatives rejected.**

- *Combined Lua that swaps `controlCharacterId` in one EVAL.* Eliminates the race but diverges from PRD's explicit guidance and adds Lua surface to test/maintain.
- *Wider WATCH scope spanning both ops.* Marginal benefit, extra code, doesn't actually serialize the two writes.

### 3.5 `AGGRO_CHANGED` emission timing (Decision 5)

**Problem.** `Damage` loops over multiple lines per attack. The script returns `wasFirstHit = true` once (on the first line that wrote the first entry); subsequent lines return `false`. FR-22 says emit `AGGRO_CHANGED` only when the flag flipped *and* no `START_CONTROL` is being emitted in the same attack.

**Choice.** Track two locals across the per-line loop and decide at the tail:

```go
firstHitObserved := false
controllerSwitched := false
for _, d := range damages {
    s, err := GetMonsterRegistry().ApplyDamage(p.t, characterId, d, m.UniqueId(), nowMs)
    if err != nil { break }
    if s.WasFirstHit { firstHitObserved = true }
    last = s
    if s.Killed { killed = true; break }
}
// existing damaged event emit
// ...
// existing controller-switch branch — sets controllerSwitched = true if it fires
// ...
if firstHitObserved && !controllerSwitched {
    p.emitAggroChanged(last.Monster, last.Monster.ControlCharacterId(), true)
}
```

When the first hit is also the hit that triggers a controller switch, `controllerSwitched = true` suppresses `AGGRO_CHANGED` because the new `START_CONTROL` already carries `controllerHasAggro: true`. When the first hit is by the existing controller (no switch), the standalone `AGGRO_CHANGED` fires.

**Alternatives rejected.**

- *Emit immediately on the line that returned `wasFirstHit`.* Can't honor FR-22 — we don't yet know whether a switch will happen.
- *Move emit into the registry layer.* The FR-22 decision needs both `wasFirstHit` and the switch decision; both live at the processor level.

---

## 4. Data flow

### 4.1 Damage path

```
atlas-channel
  └── DAMAGE command
      └── atlas-monsters Processor.Damage(id, characterId, damages, attackType)
          ├── GetMonster (load, alive check)
          ├── checkReflect (existing — unchanged)
          ├── information.GetById (existing — boss/revives lookup)
          ├── nowMs := time.Now().UnixMilli()
          ├── for each damage line:
          │   └── Registry.ApplyDamage(t, charId, d, uniqueId, nowMs)
          │       └── EVAL applyDamageScript
          │           ├── decode storedMonster
          │           ├── upsert entry by characterId (sum damage, set lastHitMs)
          │           ├── if controlCharacterId != 0 and not controllerHasAggro:
          │           │   ├── controllerHasAggro = true
          │           │   └── wasFirstHit = true
          │           └── return {wasFirstHit, monster}
          │       └── decode envelope, set DamageSummary.WasFirstHit, propagate
          ├── emit DAMAGED (existing)
          ├── if killed: emit KILLED, RemoveMonster, spawnRevives (existing)
          ├── else if characterId != current controller and DamageLeader() == characterId:
          │   ├── attackerInField check (FR-10, Decision 3)
          │   └── if in field:
          │       ├── StopControl (emits STOP_CONTROL for old controller)
          │       ├── StartControl (emits START_CONTROL with controllerHasAggro: true)
          │       └── controllerSwitched = true
          └── if firstHitObserved and not controllerSwitched:
              └── emit AGGRO_CHANGED (Decision 5)
```

### 4.2 Decay sweep path

```
tasks.Register loop fires every 1500ms
  └── MonsterAggroDecayTask.Run()
      ├── snapshot := GetMonsterRegistry().GetMonsters()  (per-tenant map)
      ├── bossCache := map[uint32]bool{}  (per-tick scope)
      └── for each tenant, for each monster:
          ├── if isBoss(monster, bossCache): skip
          ├── if len(damageEntries) == 0: skip
          ├── needsWork := false  (Go pre-filter, Decision 2 step 1)
          │   for each entry:
          │       if IsAggroIdle(entry, nowMs):
          │           needsWork = true; break
          ├── if not needsWork: continue
          ├── Registry.DecayDamageEntries(t, uniqueId, nowMs)  (Decision 2 step 2)
          │   └── EVAL decayDamageEntriesScript
          │       ├── decode storedMonster
          │       ├── for each entry:
          │       │   if (now - lastHitMs) > idleMs: damage = floor(damage * mult)
          │       │   if damage >= floor: keep; else: drop
          │       ├── controllerCleared := false
          │       ├── if no entries kept and controlCharacterId != 0:
          │       │   ├── controlCharacterId = 0
          │       │   ├── controllerHasAggro = false
          │       │   └── controllerCleared = true
          │       └── return {controllerCleared, prevControllerId, monster}
          └── if controllerCleared: emit STOP_CONTROL with prevControllerId  (Decision 2 step 3, FR-19)
```

The boss-flag cache uses `information.GetById(...).Boss()` per template-id, populated on first miss within the tick. Discarded when `Run()` returns.

---

## 5. Error handling

The damage and decay paths are best-effort: any failure logs and continues. No retries; the next damage event or decay tick re-converges.

| Failure | Handling |
|---|---|
| `ApplyDamage` Lua EVAL error | Log error, break the per-line loop, skip controller-switch and `AGGRO_CHANGED` emission for this attack. Damage already applied to earlier lines stays. |
| Envelope JSON decode error | Treat as Lua failure (above). Indicates schema drift between Go and Lua; tests should catch. |
| `DecayDamageEntries` Lua error | Log error, skip this monster for this tick. Next tick retries. |
| `attackerInField` provider error (FR-10 check fails) | Log warning, skip controller switch (fail-closed: don't grant control to an attacker we can't verify). Damage stays applied. |
| `STOP_CONTROL` / `START_CONTROL` / `AGGRO_CHANGED` emit error | Log error, no retry. State in Redis is correct; the channel will resync on the next legitimate event. |
| `information.GetById` boss lookup fails in decay task | Treat as `Boss() == false` (apply decay normally). Boss flag is best-effort metadata; misclassifying a boss for one tick is acceptable. |

---

## 6. Observability

Per PRD §8.3, debug-level logs only — no new metrics in v1.

- `Damage` controller-switch path logs old controller, new controller, and the new controller's accumulated damage.
- `Damage` `AGGRO_CHANGED` emission logs monster `uniqueId` and the controller it was sent for.
- Decay sweep logs per-monster decay events at debug level: `uniqueId`, count of entries decayed, count pruned. On controller-clear (FR-19), logs the previous controller id.
- FR-10 skip path logs the attacker character id and monster `uniqueId` at debug level.

If controller-churn or decay-sweep latency becomes a concern post-deploy, counters can be added.

---

## 7. Testing approach

### 7.1 atlas-monsters

**`monster/registry_test.go`** — extended with:

- Aggregated-entry behavior: two `ApplyDamage` calls from the same `characterId` produce one entry with summed damage and updated `lastHitMs`.
- `wasFirstHit` semantics: returns `true` on the first hit when a controller exists; `false` on subsequent hits; `false` when no controller is set.
- Decay script: entries past `AggroIdleThresholdMs` decay by `AggroDecayMultiplier`; entries below `AggroDecayFloor` are pruned; when all entries prune and a controller exists, `controllerCleared = true` and `prevControllerId` is returned.
- Legacy migration: `fromStored` consuming a `storedMonster` with multiple per-character `damageEntries` rows and no `lastHitMs` produces a single aggregated entry per character with `LastHitMs == 0`. Missing `controllerHasAggro` defaults to `false`.

**`monster/processor_test.go`** — extended with:

- Controller switch on DPS lead change (different attacker becomes leader → `STOP_CONTROL` then `START_CONTROL` with `controllerHasAggro: true`).
- No switch when the current controller takes additional damage and stays leader.
- `AGGRO_CHANGED` emitted on first hit when the controller doesn't change (e.g., the existing controller hits a fresh mob).
- `AGGRO_CHANGED` *not* emitted when the first hit also triggers a controller switch.
- FR-10: damage from an attacker not in the monster's field applies damage but does not trigger a switch.
- FR-9: when current controller is `0`, the next attacker becomes controller via a single `START_CONTROL` with no preceding `STOP_CONTROL`.

**`monster/aggro_task_test.go` (new)** — covers:

- Decay over multiple ticks: an entry that's been idle for `2 * AggroIdleThresholdMs` decays twice.
- Full-clear path: when the last entry prunes, `STOP_CONTROL` is emitted with the previous controller id; `controllerHasAggro` resets to `false`.
- Boss exemption: a monster whose template has `Boss() == true` is skipped entirely — no decay, no controller clear, even when entries are stale.
- Boss-flag cache: `information.GetById` is called once per template-id per tick, not once per monster.

The processor and task tests use the existing `emitter` injection seam (`processor.go:53-56`) to capture emitted events without spinning up Kafka. Registry tests use `miniredis` (per existing `registry_test.go` pattern).

### 7.2 atlas-channel

- New consumer test for the `AGGRO_CHANGED` flow: decoding the event body, looking up the controller's session, calling `MonsterControlWriter` with `StartControlMonsterBody(m, hasAggro)`.
- Existing `START_CONTROL` consumer test fixtures updated to include `ControllerHasAggro` in the body. Test that the value is forwarded to `StartControlMonsterBody` (replacing today's hardcoded `false`).

### 7.3 Integration — verified manually

- Spawn a mob, hit it as character A (controller assigned via existing flow), have character B out-damage A → controller swaps to B with active control.
- Walk away from a non-boss mob; after `~10s` it stops tracking and the controller is cleared; re-aggro reassigns.
- Boss monster: damage from multiple parties, walk away repeatedly, confirm controller and aggro persist until the boss dies.

---

## 8. Out of scope (carried from PRD)

Repeated here for grep-ability; not re-litigated:

- Out-of-combat HP regen.
- Puppet aggro redirect (Rogue/Cleric summon mechanic).
- Cross-channel aggro persistence.
- Tenant-configurable decay parameters.
- Status-effect interactions with aggro (taunt, stun-flips-controller, etc.).

---

## 9. Open items deferred to plan/implementation

- **Lua `cjson` encoding of empty tables.** The existing `unmarshalTolerantArray` (`registry.go:55-74`) already handles cjson encoding `{}` for empty arrays. The new `decayDamageEntriesScript` should be tested against the empty-`damageEntries` case (after pruning the last entry) to confirm round-tripping works without re-introducing the bug noted in `registry.go:50-54`.
- **Float math in Lua.** `math.floor(damage * 0.85)` is well-defined for the integer damage values we deal with, but tests should pin specific decay sequences (e.g., damage=100 → 85 → 72 → 61 → ...) to lock the exact arithmetic in case Redis/Lua versions differ across environments.
- **Spec-Task 2 read API.** PRD OQ-2 notes the future mob-skill picker will need to read DPS leader and aggro state. This design keeps `Model.ControllerHasAggro()`, `Model.DamageLeader()`, and `Model.DamageSummary()` exported and callable; no preemptive REST surface added.
