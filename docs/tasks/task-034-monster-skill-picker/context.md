# Context — Server-Side Mob Skill Picker

> Quick-reference companion to `plan.md`. Lists the files, types, and decisions an implementing agent needs to load before starting any task.

## Source artifacts

- PRD: `docs/tasks/task-034-monster-skill-picker/prd.md`
- Design: `docs/tasks/task-034-monster-skill-picker/design.md`
- Plan: `docs/tasks/task-034-monster-skill-picker/plan.md` (this folder)

## Services touched

| Service | Why |
|---|---|
| `services/atlas-monsters/atlas.com/monsters` | Picker, sweep task, repick triggers, cooldown timestamps, `UseSkill` cleanup, `m.Alive()` guard, narrowing of `UseSkill`/`UseSkillGM`/cooldown signatures, new `NEXT_SKILL_DECIDED` producer |
| `services/atlas-channel/atlas.com/channel` | New `nextSkillInbox` cache, consumer for `NEXT_SKILL_DECIDED`, MoveLife serve into `MoveMonsterAck`, `int16 → byte` bounds-check on outbound `USE_SKILL`/`USE_SKILL_FIELD` commands, narrowing of `monster.Processor.UseSkill` and `UseSkillCommandProvider` |
| `libs/atlas-constants/monster/skill.go` | Read-only — used to identify `SkillTypeAreaPoison` (4131001) for the picker's mist exclusion |
| `libs/atlas-packet/monster/clientbound/movement_ack.go` | Read-only — `MovementAck.skillId byte`, `skillLevel byte` already exist; the picker just starts populating them |

## Key files (atlas-monsters)

| File | Role |
|---|---|
| `monster/processor.go:456-561` | Existing `UseSkill`. Lines 512-517 (`prop` re-roll) deleted. Lines 553-560 (animation-delay goroutine) gain `m.Alive()` re-check + post-skill `repickAndEmit` call. Signatures `UseSkill(uniqueId uint32, characterId uint32, skillId uint16, skillLevel uint16)` → `(..., skillId byte, skillLevel byte)`; same for `UseSkillGM`. |
| `monster/processor.go:158` | Spawn flow: `repickAndEmit(uniqueId, RepickReasonSpawn)` runs *before* `START_CONTROL` emission so the controller sees a populated decision. Reordering note: today the spawn order is `Create monster → emit CREATED → StartControl → emit START_CONTROL`. The new spawn picker call goes inside `Create` after `CreateMonster` returns and before any optional `StartControl`. |
| `monster/processor.go:209-227` | `StartControl` emits `START_CONTROL`. Add `repickAndEmit(uniqueId, RepickReasonControlChange)` after the emit succeeds. |
| `monster/processor.go:244-368` | `Damage`. After successful `ApplyDamage`, compare `oldHp%` vs new HP%. If different, call `repickAndEmit(uniqueId, RepickReasonDamaged)`. Skip if killed (no need — destroyed). |
| `monster/processor.go:812-844` | `ApplyStatusEffect`. After successful registry update, if effect's status set intersects the picker-relevant set, call `repickAndEmit(uniqueId, RepickReasonStatusApplied)`. |
| `monster/processor.go:881-919` | `CancelStatusEffect` / `CancelAllStatusEffects`. After successful registry update, if any cancelled effect's status intersected the picker-relevant set, call `repickAndEmit(uniqueId, RepickReasonStatusExpired)`. |
| `monster/status_task.go:35-54` | `StatusExpirationTask.processMonsterEffects`. After emitting `STATUS_EXPIRED`, also call `repickAndEmit(uniqueId, RepickReasonStatusExpired)` if the expired effect intersected the picker-relevant set. |
| `monster/cooldown.go` | All 4 functions (`cooldownKey`, `IsOnCooldown`, `SetCooldown`, `ClearCooldowns`) gain `byte` for `skillId`. `SetCooldown` value changes from `"1"` to a millisecond expiry timestamp. New `Remaining(...) time.Duration`. |
| `monster/model.go` | Add `nextSkillDecision` private field (struct of `(skillId byte, skillLevel byte, decidedAtMs int64, nextEligibleRepickAtMs int64)`), getter `NextSkillDecision()`, helper `WithNextSkillDecision(d)`. |
| `monster/builder.go` | Mirror new field in `ModelBuilder`. Add `SetNextSkillDecision(d)`. |
| `monster/registry.go` | Add `SetNextSkillDecision(t tenant.Model, uniqueId uint32, d nextSkillDecision) (Model, error)` using `atomicUpdate`. `storedMonster` is **not** extended — decision stays in-memory only and is dropped on Redis round-trip. Reconstruction in `fromStored` returns the zero-valued sentinel decision. |
| `monster/picker.go` (NEW) | `Decision` struct, `RepickReason` enum, `pickNextSkill` pure function, `repickAndEmit` ProcessorImpl method, picker-relevant status-name set helper `isPickerRelevantStatus`. |
| `monster/picker_task.go` (NEW) | `MonsterSkillPickerSweepTask` running every 1500ms. |
| `monster/producer.go` | Add `nextSkillDecidedStatusEventProvider`. |
| `monster/kafka.go` | Add `EventMonsterStatusNextSkillDecided = "NEXT_SKILL_DECIDED"` and `statusEventNextSkillDecidedBody` struct. |
| `main.go:84-86` | Register `MonsterSkillPickerSweepTask` alongside `StatusExpirationTask` and `MonsterAggroDecayTask`. |
| `kafka/consumer/monster/kafka.go:58-66` | Narrow `useSkillCommandBody` and `useSkillFieldCommandBody` `SkillId/SkillLevel uint16 → byte`. |
| `kafka/consumer/monster/consumer.go:131-138, 217-233` | Wire the narrowed types into `handleUseSkillCommand` / `handleUseSkillFieldCommand`. The handler simply calls `p.UseSkill(c.MonsterId, c.Body.CharacterId, c.Body.SkillId, c.Body.SkillLevel)` — types already match. |

## Key files (atlas-channel)

| File | Role |
|---|---|
| `monster/inbox.go` (NEW) | Singleton typed registry. `nextSkillInbox` map keyed `(tenant.Id, uniqueId) → Decision`. Methods: `Put`, `TakeAndClear`, `Evict`. `sync.Once` init via `InitNextSkillInbox()`, `sync.RWMutex` over the inner map. |
| `kafka/message/monster/kafka.go:60-90` | Add `EventStatusNextSkillDecided = "NEXT_SKILL_DECIDED"` constant and `StatusEventNextSkillDecidedBody` struct (`SkillId byte, SkillLevel byte, DecidedAtMs int64, NextEligibleRepickAtMs int64`). |
| `kafka/consumer/monster/consumer.go` | Register handler: `handleStatusEventNextSkillDecided` → calls `nextSkillInbox.Put`. Extend `handleStatusEventDestroyed` to also call `nextSkillInbox.Evict`. |
| `monster/processor.go:48-51` | Narrow `Processor.UseSkill(..., skillId uint16, skillLevel uint16)` → `(..., skillId byte, skillLevel byte)`. |
| `monster/producer.go:33-49` | Narrow `UseSkillCommandProvider` and the body's `SkillId/SkillLevel uint16 → byte`. |
| `movement/processor.go:121-152` | `ForMonster`. Before constructing `MoveMonsterAck`, call `nextSkillInbox.TakeAndClear(t, objectId)`. If decision present and non-sentinel, write `useSkills=true, skillId, skillLevel` into the ack. The `if skillId > 0` branch (line 145) gains the `int16 → byte` bounds-check before forwarding the `USE_SKILL` Kafka command. |
| `monster/builder.go` / `monster/model.go` | Read-only — `Decision` is its own type (not on `monster.Model`). |

## Key types and signatures (after this task)

```go
// atlas-monsters monster/picker.go
type Decision struct {
    SkillId                byte
    SkillLevel             byte
    DecidedAtMs            int64
    NextEligibleRepickAtMs int64
}

func (d Decision) IsSentinel() bool { return d.SkillId == 0 }

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

func pickNextSkill(
    l logrus.FieldLogger,
    ctx context.Context,
    t tenant.Model,
    m Model,
    cooldown cooldownReader,
    rng randSource,
    nowMs int64,
) Decision

func (p *ProcessorImpl) repickAndEmit(uniqueId uint32, reason RepickReason) error

// atlas-monsters monster/cooldown.go (after narrowing)
func (r *cooldownRegistry) IsOnCooldown(ctx context.Context, t tenant.Model, monsterId uint32, skillId byte) bool
func (r *cooldownRegistry) SetCooldown(ctx context.Context, t tenant.Model, monsterId uint32, skillId byte, duration time.Duration)
func (r *cooldownRegistry) ClearCooldowns(ctx context.Context, t tenant.Model, monsterId uint32)
func (r *cooldownRegistry) Remaining(ctx context.Context, t tenant.Model, monsterId uint32, skillId byte) time.Duration

// atlas-monsters monster/processor.go (after narrowing)
func (p *ProcessorImpl) UseSkill(uniqueId uint32, characterId uint32, skillId byte, skillLevel byte)
func (p *ProcessorImpl) UseSkillGM(uniqueId uint32, skillId byte, skillLevel byte)

// atlas-channel monster/inbox.go
type Decision struct {
    SkillId                byte
    SkillLevel             byte
    DecidedAtMs            int64
    NextEligibleRepickAtMs int64
}

func InitNextSkillInbox()
func GetNextSkillInbox() *nextSkillInbox

func (r *nextSkillInbox) Put(t tenant.Model, uniqueId uint32, d Decision)
func (r *nextSkillInbox) TakeAndClear(t tenant.Model, uniqueId uint32) (Decision, bool)
func (r *nextSkillInbox) Evict(t tenant.Model, uniqueId uint32)

// atlas-channel monster/processor.go (after narrowing)
func (p *Processor) UseSkill(f field.Model, monsterId uint32, characterId uint32, skillId byte, skillLevel byte) error
```

## Pre-existing helpers and patterns to reuse

- **Immutable model + builder**: see `monster/model.go` and `monster/builder.go`. Always go through `Clone(m).Set...().Build()`. Don't mutate `Model` fields directly.
- **Registry singleton + `sync.Once`**: see `monster/cooldown.go:18-25` and `monster/registry.go:236-244`. The atlas-channel inbox uses the same shape.
- **Atomic registry update**: `r.atomicUpdate(ctx, t, uniqueId, fn)` (see `monster/registry.go`). The `SetNextSkillDecision` helper goes through the same path.
- **Status-effect lookup**: `m.HasStatusEffect("SEAL")` etc. — see `model.go:205-212`. Picker-relevant set: `SEAL`, `WEAPON_REFLECT`, `MAGIC_REFLECT`, `WEAPON_IMMUNITY`, `MAGIC_IMMUNITY`, `SEAL_SKILL`. Constants are in `libs/atlas-constants/monster/skill.go` as `TemporaryStatType*` for some, but the picker compares against the JSON status-name strings used in `Statuses()` maps.
- **Picker-relevant status-name strings** (these are the keys that appear inside `effect.Statuses()`): `SEAL`, `WEAPON_REFLECT`, `MAGIC_REFLECT`, `WEAPON_IMMUNITY`, `MAGIC_IMMUNITY`, `SEAL_SKILL`. Encode as a `map[string]struct{}` in `picker.go`.
- **Mist exclusion**: AREA_POISON has skill type ID `SkillTypeAreaPoison` (`libs/atlas-constants/monster/skill.go`). Note: `SkillCategory()` returns `SkillCategoryDebuff` for AREA_POISON (it shares the debuff bucket), so the exclusion compares against the **skill type id** (`uint16(skillId) == monster2.SkillTypeAreaPoison`), not the category string.
- **Task scheduling**: `tasks.Register(l, ctx)(taskInstance)` pattern from `main.go:83-86`. Tasks implement `SleepTime()` and `Run()`.
- **Producer**: emit via the injected `p.emit` in `ProcessorImpl` (see `processor.go:73-75`). Tests intercept by injecting a fake emitter.
- **Test patterns**: `processor_test.go`, `aggro_task_test.go`, `producer_test.go` — they wire fake emitters and registry-only state. `monster.GetMonsterRegistry()` uses Redis in production but tests typically set up a miniredis or fakeredis client.

## Key decisions (from design.md §9)

1. **Cooldown storage**: absolute expiry timestamp as the value (string-encoded int64 ms), not boolean `"1"`. Keep TTL for natural cleanup. Migration: stale `"1"` keys parse as `expiry=1` (in the past) → `Remaining` returns zero, but `IsOnCooldown` (still using `EXISTS`) returns true → skill silently no-ops on UseSkill until natural TTL evicts.
2. **Trigger plumbing**: typed `RepickReason` enum + single `repickAndEmit(uniqueId, reason)` entry point. Don't sprinkle inline picker calls.
3. **Sweep task**: dedicated `MonsterSkillPickerSweepTask`, do not fold into `MonsterAggroDecayTask`.
4. **Post-`UseSkill` re-pick timing**: re-pick from inside the animation-delay goroutine *after* `executeEffect()` returns and only if `m.Alive()` is still true at that point. For `animDelay == 0`, re-pick on the synchronous path after `executeEffect()`.
5. **Inbox shape**: singleton typed registry with `sync.Once` + `sync.RWMutex` (matches `cooldownRegistry`).
6. **Inbox file location**: `services/atlas-channel/atlas.com/channel/monster/inbox.go`.
7. **Picker placement**: new file `monster/picker.go` (separate from `processor.go`).
8. **Logging level**: `Debug` per-run; `Info` when sentinel↔non-sentinel transitions.
9. **Inbox terminology**: rename PRD's "cache" → "inbox". Add a short pattern doc at `docs/inbox-pattern.md` after the inbox is in place.

## Build & test commands

```bash
# atlas-monsters
cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./...

# atlas-channel
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...

# libs (sanity)
cd libs/atlas-packet && go build ./... && go test ./...
cd libs/atlas-constants && go build ./... && go test ./...
```

## Common gotchas

- `world.Id` is `byte`, `channel.Id` is `byte`, `_map.Id` is `uint32`. Don't substitute `string`.
- `tenant.Model.Region()` returns `string`, not a typed ID — used in cooldown key prefix.
- Registry methods often pass `tenant.Model`, not `tenant.Id`. The inbox uses `tenant.Id` *internally* (cheaper map key) but accepts `tenant.Model` at the API surface.
- The `monster.Model` in atlas-channel is a separate, lighter shape than atlas-monsters' `monster.Model`. The inbox stores its own `Decision` value, not a `monster.Model`.
- `sd.Hp() == 0` means "no HP gate" (not "dead-only") — eligibility passes regardless of HP%. Mirrors existing `processor.go:486` logic.
- Cooldown keys today are stored with value `"1"`. On the first picker run after deploy, parsing as int64 yields `1`, which is a past timestamp — `Remaining` returns zero. This is correct (eligible) for the picker; `UseSkill`'s `IsOnCooldown` still uses `EXISTS` and will block re-cast until the old TTL evicts naturally.
- The `MovementAck` packet shape is unchanged — `useSkills bool, skillId byte, skillLevel byte` already exist; today they're always written as `false, 0, 0`.

## Spec-Task forward references (NOT in scope here)

- Spec-Task 3: mist (`AREA_POISON`) executor + DoT mechanics + reflect tick. The picker's mist exclusion is a single guarded condition with a TODO comment naming Spec-Task 3.
- Spec-Task 4: boss multi-skill phase rotations, revive sequencing, HP-band-driven skill scripting. Bosses are NOT special-cased in this task.

## Definition of done (from PRD §10.4, restated)

- All 10.1 manual mob behavioral checks verified by the implementer.
- All 10.2 unit tests pass.
- `go build ./...` and `go test ./...` pass in `services/atlas-monsters`, `services/atlas-channel`, `libs/atlas-packet`, `libs/atlas-constants`.
- The redundant `prop` re-roll in `UseSkill` is gone; the `m.Alive()` guard on the animation-delay goroutine is in place; the picker AREA_POISON exclusion is documented in code with the Spec-Task 3 TODO.
- `docs/inbox-pattern.md` describes the inbox pattern with the atlas-channel `nextSkillInbox` as the reference example.
