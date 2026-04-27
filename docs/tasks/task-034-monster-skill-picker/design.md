# Server-Side Mob Skill Picker — Design

**PRD:** `docs/tasks/task-034-monster-skill-picker/prd.md`
**Status:** Draft, design phase
**Created:** 2026-04-26

## 1. Architecture Summary

atlas-monsters becomes the authoritative chooser of which mob skill, if any, fires on the next controller tick. Each significant state change runs `pickNextSkill`, which evaluates the executor's eligibility gates and rolls each skill's `prop` independently. The decision is stored on the in-memory `monster.Model`, then emitted as `NEXT_SKILL_DECIDED` on `EVENT_TOPIC_MONSTER_STATUS`. atlas-channel mirrors the latest decision into a per-monster **inbox** (single-use prediction handoff), and the next `MoveLife` packet from the controller serves it into the outbound `MoveMonsterAck`. The controller's client echoes the skill back on the following tick, at which point the existing `UseSkill` Kafka path validates and applies it.

Three smaller concerns are bundled in:

- `UseSkill`'s `prop` re-roll is removed (the picker is now the sole authority on `prop`).
- The animation-delay goroutine gets an `m.Alive()` re-check before applying its effect.
- `UseSkill` / `UseSkillGM` / cooldown registry signatures narrow from `uint16` to `byte` for `skillId` / `skillLevel`.

## 2. atlas-monsters changes

### 2.1 Picker — `monster/picker.go` (new)

`pickNextSkill(p ProcessorImpl)(m Model) (Decision, time.Time)` is a pure function (no mutations). It iterates `information.Model.Skills()`, narrows each `(uint32 id, uint32 level)` to `byte`, and runs the eligibility gates from PRD §FR-2 in order. The eligibility result for each skill is one of:

- **eligible** — passes all gates; rolled with `rand.Intn(100) < int(sd.Prop())`. First successful roll wins.
- **cooldown-gated** — fails only because the cooldown is active. Contributes its expiry timestamp to `nextEligibleRepickAtMs` (minimum across all cooldown-gated skills).
- **ineligible (other)** — fails for any non-cooldown reason (HP, MP, status, AREA_POISON exclusion, byte overflow). Does not contribute to `nextEligibleRepickAtMs`.

Returns a `Decision{SkillId byte, SkillLevel byte, NextEligibleRepickAtMs int64}` plus a `decidedAt time.Time` for the event.

A thin method on `ProcessorImpl`:

```go
type RepickReason string

const (
    RepickReasonSpawn         RepickReason = "spawn"
    RepickReasonPostUseSkill  RepickReason = "post_use_skill"
    RepickReasonDamaged       RepickReason = "damaged"
    RepickReasonStatusApplied RepickReason = "status_applied"
    RepickReasonStatusExpired RepickReason = "status_expired"
    RepickReasonControlChange RepickReason = "control_change"
    RepickReasonSweep         RepickReason = "sweep"
)

func (p *ProcessorImpl) repickAndEmit(uniqueId uint32, reason RepickReason) error
```

`repickAndEmit` reads the monster from the registry, runs the picker, writes the decision back via a registry method `SetNextSkillDecision(uniqueId, decision)`, and emits the event. Always emits, even if the decision is unchanged or sentinel. Logs at debug level (per-run) and at info level on sentinel↔non-sentinel transitions.

### 2.2 Cooldown registry — `monster/cooldown.go` (modified)

Storage shape changes from boolean+TTL to **absolute expiry timestamp+TTL**. The key still has a TTL (Redis garbage-collects naturally); the value is the expiry millis as a string. New method:

```go
func (r *cooldownRegistry) Remaining(ctx context.Context, t tenant.Model, monsterId uint32, skillId byte) time.Duration
```

Implementation: `GET` the key, parse the value as int64, return `expiry - now` (clamped to zero). On parse failure, missing key, or expiry-in-the-past, returns zero (treat as eligible). `IsOnCooldown` continues to use `EXISTS` for the simpler boolean answer.

`SetCooldown` switches to writing the value with `SET key <expiryMs> PX <durationMs>`. `ClearCooldowns` is unchanged. Signature narrows `skillId uint16 → byte` in all methods.

Migration: pre-deploy keys hold the value `"1"`. After deploy, the picker reads them, parses as int64 (succeeds with value `1`, which is in the past), treats as eligible. UseSkill's `IsOnCooldown` path still returns true (key exists), so the skill silently no-ops on re-cast attempt until the old TTL evicts (within seconds for typical intervals). No corruption.

### 2.3 Monster model — `monster/model.go`, `monster/builder.go` (modified)

Add private field `nextSkillDecision` to `Model` with `(skillId byte, skillLevel byte, decidedAtMs int64, nextEligibleRepickAtMs int64)`. Builder method `SetNextSkillDecision`. Initial value: zero struct (sentinel).

Add registry method `SetNextSkillDecision(t, uniqueId, decision)`. Existing registry pattern; `sync.RWMutex` write side.

### 2.4 Trigger plumbing

Six call sites add `p.repickAndEmit(uniqueId, RepickReason...)`:

| Site | File | Reason |
|---|---|---|
| Spawn (after registry commit, before `START_CONTROL` emit) | `processor.go` create flow | `RepickReasonSpawn` |
| Post-`UseSkill` after `executeEffect()` returns, gated by `m.Alive()` re-check | `processor.go:553-560` | `RepickReasonPostUseSkill` |
| Damage handler, after HP update, only when `oldHp% != newHp%` | existing damage path | `RepickReasonDamaged` |
| Status apply (only for SEAL / WEAPON_REFLECT / MAGIC_REFLECT / WEAPON_IMMUNITY / MAGIC_IMMUNITY / SEAL_SKILL) | status apply path | `RepickReasonStatusApplied` |
| Status expire (same filter set) | status expire path | `RepickReasonStatusExpired` |
| `START_CONTROL` emit (initial assignment + task-033 controller switch) | controller change path | `RepickReasonControlChange` |

The status filter is a small package-level set of status names; helper `isPickerRelevantStatus(name string) bool`. Other statuses do not trigger the picker.

### 2.5 Sweep task — `monster/picker_task.go` (new)

`MonsterSkillPickerSweepTask` mirrors `MonsterAggroDecayTask`: 1500ms interval, walks `GetMonsterRegistry().GetMonsters()`, per-monster precondition check `len(skills) > 0 && nextEligibleRepickAtMs > 0 && nextEligibleRepickAtMs <= now`, calls `repickAndEmit(uniqueId, RepickReasonSweep)`. Registered in the task wiring alongside the existing two tasks.

### 2.6 Producer — `monster/producer.go`, `monster/kafka.go` (modified)

Add:

```go
EventMonsterStatusNextSkillDecided = "NEXT_SKILL_DECIDED"

type statusEventNextSkillDecidedBody struct {
    SkillId                byte  `json:"skillId"`
    SkillLevel             byte  `json:"skillLevel"`
    DecidedAtMs            int64 `json:"decidedAtMs"`
    NextEligibleRepickAtMs int64 `json:"nextEligibleRepickAtMs"`
}

func nextSkillDecidedStatusEventProvider(m Model, decision Decision, decidedAtMs int64) model.Provider[[]kafka.Message]
```

Partition key: monster `uniqueId` (matches existing per-monster ordering convention).

### 2.7 `UseSkill` cleanup

- Lines 512-517 (`prop` re-roll): **deleted**.
- Lines 553-560 (animation delay goroutine): re-fetch monster from registry by `uniqueId`, skip `executeEffect()` if not present or `!Alive()`. After `executeEffect()` returns (or after the synchronous path when `animDelay == 0`), call `p.repickAndEmit(uniqueId, RepickReasonPostUseSkill)`.
- `Processor.UseSkill` and `UseSkillGM` signatures narrow `skillId/skillLevel uint16 → byte`. The producer side (atlas-channel MoveLife handler and any GM tooling) narrows `int16 → byte` with bounds-check; out-of-range drops the command with a warning.

## 3. atlas-channel changes

### 3.1 Inbox — `monster/inbox.go` (new)

Singleton typed registry, `sync.Once` init, `sync.RWMutex` over `map[tenant.Id]map[uint32]Decision`. Methods:

```go
func (r *nextSkillInbox) Put(t tenant.Model, uniqueId uint32, d Decision)
func (r *nextSkillInbox) TakeAndClear(t tenant.Model, uniqueId uint32) (Decision, bool)
func (r *nextSkillInbox) Evict(t tenant.Model, uniqueId uint32)
```

`Put` overwrites (last-writer-wins). `TakeAndClear` returns the entry if present, removing it (single-use serve). `Evict` is for `MONSTER_DESTROYED`.

The `Decision` value type carries `(SkillId, SkillLevel byte)` plus the metadata fields from the event. Sentinel detection is `SkillId == 0` (mirrors PRD §5.1).

### 3.2 Consumer — `kafka/consumer/monster/consumer.go` (modified)

Add `handleStatusEventNextSkillDecided` alongside the existing handlers. On the matching event type, call `nextSkillInbox.Put`. Existing `handleStatusEventDestroyed` gains a sibling call to `nextSkillInbox.Evict`.

The corresponding event body type lives in `kafka/message/monster/kafka.go`:

```go
const EventStatusNextSkillDecided = "NEXT_SKILL_DECIDED"

type StatusEventNextSkillDecidedBody struct {
    SkillId                byte  `json:"skillId"`
    SkillLevel             byte  `json:"skillLevel"`
    DecidedAtMs            int64 `json:"decidedAtMs"`
    NextEligibleRepickAtMs int64 `json:"nextEligibleRepickAtMs"`
}
```

### 3.3 MoveLife serve — `movement/processor.go:ForMonster` (modified)

Before constructing the `MoveMonsterAck`, call `nextSkillInbox.TakeAndClear(t, objectId)`. If present and non-sentinel, write `useSkills=true, skillId=d.SkillId, skillLevel=d.SkillLevel` into the ack. Otherwise pass `false, 0, 0` as today.

The broadcast `MoveMonster` packet (sent to other clients) continues to forward the inbound `skillId/skillLevel` from the serverbound MoveLife verbatim — **not** the predicted skill. The prediction only goes to the controller's ack; the broadcast picks up the real cast on the *next* MoveLife.

### 3.4 `int16 → byte` bounds-check on outbound `USE_SKILL` command

In the MoveLife handler path (the existing `if skillId > 0` branch at `movement/processor.go:145`), narrow with a guard: if `skillId < 0 || skillId > 255 || skillLevel < 0 || skillLevel > 255`, log a warning and **drop** the command without forwarding. Same guard applies to the GM `USE_SKILL_FIELD` producer site if it exists.

## 4. Data flow

Annotated end-to-end trace for "Stirge casts DARKNESS within ~10s of engagement":

1. Player's first hit lands → atlas-monsters' damage handler updates HP → HP% bucket changes → `repickAndEmit(uniqueId, RepickReasonDamaged)` runs.
2. Picker iterates Stirge's skill list; DARKNESS passes all gates and rolls successfully on `prop`.
3. Decision is stored on `Model` and emitted as `NEXT_SKILL_DECIDED{skillId=DARKNESS, skillLevel=1, decidedAtMs=..., nextEligibleRepickAtMs=...}`.
4. atlas-channel's consumer receives the event, writes `(tenantId, uniqueId) → Decision` into the inbox.
5. Stirge's controller (the player's client) sends the next `MoveLife`. atlas-channel's MoveLife handler calls `inbox.TakeAndClear` → finds the decision → writes it into `MoveMonsterAck.skillId/skillLevel/useSkills`. Inbox entry is now empty.
6. The client receives the ack, plays the cast animation, sends a follow-up `MoveLife` with `skillId=DARKNESS, skillLevel=1` in the inbound packet.
7. atlas-channel's MoveLife handler forwards a `USE_SKILL` Kafka command (with the `int16 → byte` guard) to atlas-monsters.
8. atlas-monsters' `UseSkill` re-validates eligibility (cooldown, MP, HP%, status), deducts MP, sets the cooldown via `SET key <expiryMs> PX <duration>`, dispatches to the executor (`executeDebuff` for DARKNESS), and after `executeEffect()` returns, calls `repickAndEmit(uniqueId, RepickReasonPostUseSkill)`. The new decision typically excludes DARKNESS because it's now on cooldown.

## 5. Failure modes

Per PRD §8.5, plus the design-specific clarifications:

- **Inbox stale on serve**: handled by `UseSkill`'s eligibility re-check (defense-in-depth). Skill silently no-ops; no MP or cooldown consumed.
- **Cooldown key parse-failure**: `Remaining` returns zero, picker treats skill as eligible. `UseSkill`'s `IsOnCooldown` (still using `EXISTS`) keeps it from actually firing during the old-format-key transition window.
- **Two MoveLifes before atlas-monsters re-emits**: first serves the prediction, second falls through to default no-skill. Acceptable by design (single-use serve).
- **atlas-monsters restart**: monsters rehydrate from Redis, picker re-runs as part of rehydration, fresh decisions repopulate inboxes.
- **atlas-channel restart**: inbox is empty until atlas-monsters' next emission cycle (≤1500ms via sweep, immediate on next state-change trigger).

## 6. Testing strategy

PRD §10.2 enumerates the unit tests. Design-specific notes:

- **Picker tests** parameterize a fake cooldown registry (returns canned `Remaining` values) and a fixed-seed RNG via a `randSource` interface injected into `pickNextSkill`. Tests cover: empty skill list, single-skill HP-gated, single-skill cooldown-gated, single-skill MP-gated, sealed monster, reflect already active, AREA_POISON exclusion, byte-overflow guard, prop-roll determinism, `nextEligibleRepickAtMs` minimum across multiple cooldown-gated skills.
- **`m.Alive()` guard test** uses the existing `MonsterRegistry` fake; constructs a monster, marks it dead, schedules an animation-delayed effect, asserts no executor call after the wake-up.
- **Inbox tests** in atlas-channel: `Put` followed by `TakeAndClear` returns the value and clears; second `TakeAndClear` returns `(_, false)`; `Evict` on a present key clears it; multi-tenant isolation (same `uniqueId` in two tenants are independent entries).
- **Cooldown registry tests**: `SetCooldown` followed by `Remaining` returns the expected `time.Duration` (within a tolerance); `Remaining` on a missing key returns zero; `Remaining` on an expired key (manually `SET` with a past timestamp) returns zero; `IsOnCooldown` still returns true while the key exists.
- **MoveLife `int16 → byte` guard test**: out-of-range inbound values cause the producer to skip emit and log a warning.

## 7. Guideline addendum

A new short doc at `docs/inbox-pattern.md` describes the **inbox pattern**:

> An inbox is an in-process, per-key map that holds a single-use handoff between an asynchronous producer and a synchronous consumer in the same process. Distinct from a registry (long-lived state, multi-read), distinct from a cache (look-aside backed by a source of truth). Use an inbox when an external decision needs to influence the next packet/response that fires for a given key, and the decision arrives at a different time than the consumption point. `Put` is last-writer-wins; `TakeAndClear` is the standard read; entries are evicted by an explicit lifecycle event (e.g. resource destruction). The atlas-channel `nextSkillInbox` (mob skill prediction handoff) is the reference example.

## 8. Open items deferred

None for implementation. Spec-Task 3 (mist / DoT executors) and Spec-Task 4 (boss multi-skill rotations) are explicit non-goals per PRD §2.

## 9. Decision log

Choices made during design phase (PRD-deferred or newly identified):

| # | Decision | Chosen | Alternatives considered |
|---|---|---|---|
| 1 | How to compute `nextEligibleRepickAtMs` | C: store absolute expiry timestamp as cooldown key value; keep TTL | A: add Redis `PTTL` query; B: in-memory mirror map |
| 2 | Re-pick trigger plumbing | B: `repickAndEmit(uniqueId, reason)` with typed reason enum | A: inline calls without reason; C: channel-fed worker goroutine |
| 3 | Sweep task wiring (PRD §9 open) | A: dedicated `MonsterSkillPickerSweepTask` | B: fold into `MonsterAggroDecayTask` |
| 4 | Post-`UseSkill` re-pick timing | B: re-pick from inside animation-delay goroutine after `executeEffect()` returns, gated by `m.Alive()` | A: re-pick at end of `UseSkill` before delay; C: both |
| 5 | atlas-channel inbox shape | A: singleton typed registry (`sync.Once` + `sync.RWMutex`) | B: `sync.Map`; C: instance-held value |
| 6 | Inbox file location | B: new file inside existing atlas-channel `monster/` package | A: new sibling package; C: under `kafka/consumer/monster/` |
| 7 | Picker placement (PRD §9 open) | New file `monster/picker.go` | Methods on `ProcessorImpl` in `processor.go` |
| 8 | Logging level (PRD §9 open) | Debug per-run; info on sentinel↔non-sentinel transitions | Info per-run |
| 9 | Inbox terminology | Rename `nextSkillCache` → `nextSkillInbox` and document the pattern in guidelines | Keep PRD's "cache" terminology |
