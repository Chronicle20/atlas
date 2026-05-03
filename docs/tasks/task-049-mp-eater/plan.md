# MP Eater Passive Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the MP Eater passive (Fire/Poison Wizard `2100000`, Ice/Lightning Wizard `2200000`, Cleric `2300000`) so every magic attack — including Heal vs. undead — has a per-monster chance to drain MP from a non-boss target back to the caster, accompanied by a SKILL_SPECIAL visual.

**Architecture:** Authority is split. atlas-channel evaluates the proc inline in `processAttack` (skill resolution, RNG roll, amount computation), then emits a `DRAIN_MP` command on `COMMAND_TOPIC_MONSTER`. atlas-monsters re-checks guards (boss/MaxMp/Mp), `DeductMp` clamps at zero, and emits a `MP_CHANGED` status event with a `Reason` discriminator. atlas-channel consumes the event, refunds caster MP via the existing `character.Processor.ChangeMP`, and broadcasts `CharacterSkillSpecialEffect{,Foreign}Body` packets. RNG is injected as a parameter to a pure helper, mirroring the `snapshotVenomDamagePerTick` seam already in the same file.

**Tech Stack:** Go, Kafka (segmentio/kafka-go), JSON:API REST, immutable models with Builder, curried producer/consumer registration. Co-located `_test.go` files using the standard library `testing` package.

---

## Conventions used in this plan

- "Run tests" steps cite an exact `go test` invocation with `-run` regex pinning the new test, plus the **expected outcome**.
- Steps that introduce code show the **complete code block** to add (not "similar to X").
- One commit per task (typically). Some tasks intentionally combine TDD-test and implementation commits when the failing-test cycle is self-contained.
- Symbol corrections relative to the design doc:
  - Design says `skill.Registry` — actual symbol is `skill.Skills` (a `map[Id]Skill`). The plan uses `skill.Skills`.
  - Design says `EventStatusMpChanged` for the channel-side const — kept as `EventStatusMpChanged` in the channel's kafka package; the atlas-monsters package mirrors this with `EventMonsterStatusMpChanged` (matching the existing `EventMonsterStatus*` naming there).

---

## File map

**atlas-channel — modified:**
- `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go` — add `Prop()`, `X()` accessors.
- `services/atlas-channel/atlas.com/channel/monster/model.go` — add `maxMp`, `MaxMp()`.
- `services/atlas-channel/atlas.com/channel/monster/builder.go` — add `maxMp` field + `SetMaxMp`.
- `services/atlas-channel/atlas.com/channel/monster/rest.go` — wire `MaxMp` in `Extract`.
- `services/atlas-channel/atlas.com/channel/socket/handler/effects.go` — add `AnnounceSkillSpecial` / `AnnounceForeignSkillSpecial`.
- `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go` — add `CommandTypeDrainMp`, `EventStatusMpChanged`, `MpChangeReasonMpEater`, `DrainMpCommandBody`, `StatusEventMpChangedBody`.
- `services/atlas-channel/atlas.com/channel/monster/producer.go` — add `DrainMpCommandProvider`.
- `services/atlas-channel/atlas.com/channel/monster/processor.go` — add `Processor.DrainMp`.
- `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go` — add `handleStatusEventMpChanged` + register.
- `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` — add helpers + `mpEaterTryProc` + call site, remove `// TODO Apply MPEater`.

**atlas-channel — created (test files):**
- `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_mp_eater_test.go`

**atlas-monsters — modified:**
- `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go` — add `CommandTypeDrainMp`, `drainMpCommandBody`.
- `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go` — add `handleDrainMpCommand` + register.
- `services/atlas-monsters/atlas.com/monsters/monster/kafka.go` — add `EventMonsterStatusMpChanged`, `MpChangeReasonMpEater`, `statusEventMpChangedBody`.
- `services/atlas-monsters/atlas.com/monsters/monster/producer.go` — add `mpChangedStatusEventProvider`.
- `services/atlas-monsters/atlas.com/monsters/monster/processor.go` — add `DrainMp` to `Processor` interface and `ProcessorImpl`.

**atlas-monsters — created (test files):**
- Tests appended to `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go` (or new `drain_mp_test.go` in the same package if the existing file is large; pick whichever the executing agent finds cleaner — both are fine).

**Shared docs:**
- `docs/TODO.md` — check off `Apply MPEater`.

---

## Task 1: Add `Prop()` and `X()` accessors on `effect.Model`

Pure mechanical accessor addition. The fields are already private on the
struct.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go`

- [ ] **Step 1: Add the accessors**

Open `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go` and
add these two methods at the bottom of the file (after `RB()` at line 122):

```go
// Prop returns the proc-chance attribute (0.0–1.0). Used by passives like
// MP Eater to roll on each affected monster.
func (m Model) Prop() float64 {
	return m.prop
}

// X returns the integer X attribute (often used as a percent or
// multiplier; for MP Eater it is the absorb percent of monster MaxMp).
func (m Model) X() int16 {
	return m.x
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./data/skill/effect/...
```

Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/data/skill/effect/model.go
git commit -m "feat(atlas-channel): expose Prop and X on skill effect.Model"
```

---

## Task 2: Add `MaxMp` to channel `monster.Model`

The REST payload already carries `MaxMp` (`rest.go:30`); the in-memory model
drops it on extraction. Restore the round-trip.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/monster/model.go`
- Modify: `services/atlas-channel/atlas.com/channel/monster/builder.go`
- Modify: `services/atlas-channel/atlas.com/channel/monster/rest.go`
- Test: `services/atlas-channel/atlas.com/channel/monster/builder_test.go` (existing — extend)

- [ ] **Step 1: Write the failing test**

Open `services/atlas-channel/atlas.com/channel/monster/builder_test.go` and
append:

```go
func TestModelBuilder_SetMaxMp(t *testing.T) {
	f := field.NewBuilder(1, 1, 100000000).SetInstance(uuid.Nil).Build()
	m, err := NewModelBuilder(42, f, 9300000).SetMaxMp(500).SetMp(200).Build()
	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}
	if m.MaxMp() != 500 {
		t.Fatalf("MaxMp() = %d; want 500", m.MaxMp())
	}
	if m.Mp() != 200 {
		t.Fatalf("Mp() = %d; want 200", m.Mp())
	}
}

func TestCloneModel_PreservesMaxMp(t *testing.T) {
	f := field.NewBuilder(1, 1, 100000000).SetInstance(uuid.Nil).Build()
	original, err := NewModelBuilder(42, f, 9300000).SetMaxMp(500).Build()
	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}
	cloned, err := CloneModel(original).Build()
	if err != nil {
		t.Fatalf("Clone Build() returned error: %v", err)
	}
	if cloned.MaxMp() != 500 {
		t.Fatalf("Cloned MaxMp() = %d; want 500", cloned.MaxMp())
	}
}
```

If `field` and `uuid` are not yet imported in this test file, add them:

```go
import (
	// ... existing imports
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/google/uuid"
)
```

- [ ] **Step 2: Run the test to verify it fails to compile**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./monster/ -run TestModelBuilder_SetMaxMp
```

Expected: **FAIL** with `m.MaxMp undefined` and/or `b.SetMaxMp undefined`.

- [ ] **Step 3: Add `maxMp` to the model**

Open `services/atlas-channel/atlas.com/channel/monster/model.go`. In the `Model`
struct (lines 25-40), add `maxMp` immediately after `mp`:

```go
type Model struct {
	field              field.Model
	uniqueId           uint32
	maxHp              uint32
	hp                 uint32
	mp                 uint32
	maxMp              uint32
	monsterId          uint32
	controlCharacterId uint32
	controllerHasAggro bool
	x                  int16
	y                  int16
	fh                 int16
	stance             byte
	team               int8
	statusEffects      []StatusEffectEntry
}
```

Then add the accessor near the existing `Mp()` / `MaxHp()` (after `MaxHp()` at
line 120):

```go
func (m Model) MaxMp() uint32 {
	return m.maxMp
}
```

- [ ] **Step 4: Add `maxMp` to the builder**

Open `services/atlas-channel/atlas.com/channel/monster/builder.go`. Add `maxMp`
to the struct (after `mp`):

```go
type modelBuilder struct {
	field              field.Model
	uniqueId           uint32
	maxHp              uint32
	hp                 uint32
	mp                 uint32
	maxMp              uint32
	monsterId          uint32
	controlCharacterId uint32
	x                  int16
	y                  int16
	fh                 int16
	stance             byte
	team               int8
	statusEffects      []StatusEffectEntry
}
```

In `CloneModel` (line 39), copy the field:

```go
func CloneModel(m Model) *modelBuilder {
	return &modelBuilder{
		field:              m.field,
		uniqueId:           m.uniqueId,
		maxHp:              m.maxHp,
		hp:                 m.hp,
		mp:                 m.mp,
		maxMp:              m.maxMp,
		monsterId:          m.monsterId,
		controlCharacterId: m.controlCharacterId,
		x:                  m.x,
		y:                  m.y,
		fh:                 m.fh,
		stance:             m.stance,
		team:               m.team,
		statusEffects:      m.statusEffects,
	}
}
```

Add the setter immediately after `SetMp` (line 67):

```go
func (b *modelBuilder) SetMaxMp(maxMp uint32) *modelBuilder {
	b.maxMp = maxMp
	return b
}
```

In `Build()` (line 103), include `maxMp` in the returned `Model`:

```go
return Model{
	field:              b.field,
	uniqueId:           b.uniqueId,
	maxHp:              b.maxHp,
	hp:                 b.hp,
	mp:                 b.mp,
	maxMp:              b.maxMp,
	monsterId:          b.monsterId,
	controlCharacterId: b.controlCharacterId,
	x:                  b.x,
	y:                  b.y,
	fh:                 b.fh,
	stance:             b.stance,
	team:               b.team,
	statusEffects:      b.statusEffects,
}, nil
```

- [ ] **Step 5: Wire `Extract` to populate `maxMp`**

Open `services/atlas-channel/atlas.com/channel/monster/rest.go`. In the
`Extract` return value (line 77), add `maxMp: m.MaxMp,` next to `mp: m.Mp,`:

```go
return Model{
	uniqueId:           uint32(id),
	field:              field.NewBuilder(m.WorldId, m.ChannelId, m.MapId).SetInstance(m.Instance).Build(),
	maxHp:              m.MaxHp,
	hp:                 m.Hp,
	mp:                 m.Mp,
	maxMp:              m.MaxMp,
	monsterId:          m.MonsterId,
	controlCharacterId: m.ControlCharacterId,
	controllerHasAggro: m.ControllerHasAggro,
	x:                  m.X,
	y:                  m.Y,
	fh:                 m.Fh,
	stance:             m.Stance,
	team:               m.Team,
	statusEffects:      ses,
}, nil
```

- [ ] **Step 6: Run the test to verify it passes**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./monster/ -run "TestModelBuilder_SetMaxMp|TestCloneModel_PreservesMaxMp"
```

Expected: **PASS** for both tests.

- [ ] **Step 7: Run the full monster package tests to confirm no regression**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./monster/...
```

Expected: **PASS** for the whole package.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/monster/model.go services/atlas-channel/atlas.com/channel/monster/builder.go services/atlas-channel/atlas.com/channel/monster/rest.go services/atlas-channel/atlas.com/channel/monster/builder_test.go
git commit -m "feat(atlas-channel): track monster MaxMp in live snapshot"
```

---

## Task 3: Add `AnnounceSkillSpecial` / `AnnounceForeignSkillSpecial` helpers

Mirror `AnnounceSkillUse` for the SKILL_SPECIAL effect mode. The packet builders
already exist in `libs/atlas-packet/character/effect_body.go`.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/effects.go`

- [ ] **Step 1: Append the helpers**

Open `services/atlas-channel/atlas.com/channel/socket/handler/effects.go` and
append the following at the bottom of the file (after `AnnounceForeignSkillUse`
at line 38):

```go
// AnnounceSkillSpecial broadcasts the SKILL_SPECIAL CharacterEffect to the
// caster's own session. Used by passive procs (e.g., MP Eater) to play the
// skill's "special" visual without re-broadcasting a full skill-use cast.
func AnnounceSkillSpecial(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(skillId uint32) model2.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(skillId uint32) model2.Operator[session.Model] {
		return func(wp writer.Producer) func(skillId uint32) model2.Operator[session.Model] {
			return func(skillId uint32) model2.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(charcb.CharacterEffectWriter)(charpkt.CharacterSkillSpecialEffectBody(skillId))
			}
		}
	}
}

// AnnounceForeignSkillSpecial is the same broadcast targeted at other sessions
// on the caster's map.
func AnnounceForeignSkillSpecial(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, skillId uint32) model2.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, skillId uint32) model2.Operator[session.Model] {
		return func(wp writer.Producer) func(characterId uint32, skillId uint32) model2.Operator[session.Model] {
			return func(characterId uint32, skillId uint32) model2.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(charcb.CharacterEffectForeignWriter)(charpkt.CharacterSkillSpecialEffectForeignBody(characterId, skillId))
			}
		}
	}
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./socket/handler/...
```

Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/effects.go
git commit -m "feat(atlas-channel): add AnnounceSkillSpecial broadcaster helpers"
```

---

## Task 4: Define `DRAIN_MP` / `MP_CHANGED` wire types in atlas-channel

The kafka message package on the channel side carries the exported types both
the producer (this service) and any future event consumer rely on.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go`

- [ ] **Step 1: Add command-side constants and body**

Open `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go`.
In the const block at lines 10-18, append `CommandTypeDrainMp`:

```go
const (
	EnvCommandTopic           = "COMMAND_TOPIC_MONSTER"
	CommandTypeDamage         = "DAMAGE"
	CommandTypeDamageFriendly = "DAMAGE_FRIENDLY"
	CommandTypeApplyStatus    = "APPLY_STATUS"
	CommandTypeCancelStatus   = "CANCEL_STATUS"
	CommandTypeUseSkill       = "USE_SKILL"
	CommandTypeUseBasicAttack = "USE_BASIC_ATTACK"
	CommandTypeDrainMp        = "DRAIN_MP"
)
```

After the `UseBasicAttackCommandBody` struct (line 71), append:

```go
// DrainMpCommandBody asks atlas-monsters to deduct MP from a monster
// because of a player passive. atlas-monsters re-checks Boss / MaxMp /
// current Mp guards and clamps the deduction at zero. On a non-zero
// drain it emits a MP_CHANGED status event with Reason set so the
// channel can refund the caster's MP and play the visual.
type DrainMpCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	SkillId     uint32 `json:"skillId"`
	Amount      uint32 `json:"amount"`
}
```

- [ ] **Step 2: Add status-event constants and body**

In the `EnvEventTopicStatus` const block (lines 73-93), append the new event
type and the reason constant:

```go
const (
	EnvEventTopicStatus = "EVENT_TOPIC_MONSTER_STATUS"

	EventStatusCreated          = "CREATED"
	EventStatusDestroyed        = "DESTROYED"
	EventStatusStartControl     = "START_CONTROL"
	EventStatusStopControl      = "STOP_CONTROL"
	EventStatusDamaged          = "DAMAGED"
	EventStatusKilled           = "KILLED"
	EventStatusEffectApplied    = "STATUS_APPLIED"
	EventStatusEffectExpired    = "STATUS_EXPIRED"
	EventStatusEffectCancelled  = "STATUS_CANCELLED"
	EventStatusDamageReflected  = "DAMAGE_REFLECTED"
	EventStatusAggroChanged     = "AGGRO_CHANGED"
	EventStatusNextSkillDecided = "NEXT_SKILL_DECIDED"
	EventStatusMpChanged        = "MP_CHANGED"

	DamageSourceCharacterAttack = "CHARACTER_ATTACK"
	DamageSourceMonsterAttack   = "MONSTER_ATTACK"
	DamageSourceDamageOverTime  = "DAMAGE_OVER_TIME"
	DamageSourceHeal            = "HEAL"

	MpChangeReasonMpEater = "MP_EATER"
)
```

After the `StatusEventNextSkillDecidedBody` struct (line 194), append:

```go
// StatusEventMpChangedBody is the return event for any monster MP
// mutation whose Reason atlas-channel needs to react to. v1 only emits
// Reason = MpChangeReasonMpEater; future passives (e.g., Magic Guard
// refund, Drain MP) will share the channel by setting a new Reason.
type StatusEventMpChangedBody struct {
	CharacterId    uint32 `json:"characterId"`
	SkillId        uint32 `json:"skillId"`
	Reason         string `json:"reason"`
	Amount         uint32 `json:"amount"`
	MonsterMpAfter uint32 `json:"monsterMpAfter"`
}
```

- [ ] **Step 3: Verify it compiles**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./kafka/message/monster/...
```

Expected: no output (success).

- [ ] **Step 4: Run existing kafka package tests**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./kafka/message/monster/...
```

Expected: **PASS** (no test changes; just confirming we did not break anything).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go
git commit -m "feat(atlas-channel): add DRAIN_MP command and MP_CHANGED event types"
```

---

## Task 5: Define `DRAIN_MP` wire types in atlas-monsters consumer

Mirror Task 4 on the consumer side. atlas-monsters uses unexported types per
the existing convention.

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go`

- [ ] **Step 1: Add command type constant and body**

Open `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go`.
In the const block at lines 10-24, append `CommandTypeDrainMp`:

```go
const (
	EnvCommandTopic              = "COMMAND_TOPIC_MONSTER"
	CommandTypeDamage            = "DAMAGE"
	CommandTypeDamageFriendly    = "DAMAGE_FRIENDLY"
	CommandTypeApplyStatus       = "APPLY_STATUS"
	CommandTypeCancelStatus      = "CANCEL_STATUS"
	CommandTypeUseSkill          = "USE_SKILL"
	CommandTypeUseBasicAttack    = "USE_BASIC_ATTACK"
	CommandTypeApplyStatusField  = "APPLY_STATUS_FIELD"
	CommandTypeCancelStatusField = "CANCEL_STATUS_FIELD"
	CommandTypeUseSkillField     = "USE_SKILL_FIELD"
	CommandTypeDestroyField      = "DESTROY_FIELD"
	CommandTypeDrainMp           = "DRAIN_MP"

	EnvCommandTopicMovement = "COMMAND_TOPIC_MONSTER_MOVEMENT"
)
```

After the `useBasicAttackCommandBody` struct (line 80), append:

```go
type drainMpCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	SkillId     uint32 `json:"skillId"`
	Amount      uint32 `json:"amount"`
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd services/atlas-monsters/atlas.com/monsters && go build ./kafka/consumer/monster/...
```

Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go
git commit -m "feat(atlas-monsters): add DRAIN_MP command type"
```

---

## Task 6: Define `MP_CHANGED` wire types in atlas-monsters monster package

The monster package owns the status-event constants. The Reason discriminator
lives here too.

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/kafka.go`

- [ ] **Step 1: Add the event constant and reason**

Open `services/atlas-monsters/atlas.com/monsters/monster/kafka.go`. In the
const block at lines 16-28 (the `EventMonsterStatus*` block), append:

```go
const (
	EnvEventTopicMonsterStatus = "EVENT_TOPIC_MONSTER_STATUS"

	EventMonsterStatusCreated          = "CREATED"
	EventMonsterStatusDestroyed        = "DESTROYED"
	EventMonsterStatusStartControl     = "START_CONTROL"
	EventMonsterStatusStopControl      = "STOP_CONTROL"
	EventMonsterStatusDamaged          = "DAMAGED"
	EventMonsterStatusKilled           = "KILLED"
	EventMonsterStatusEffectApplied    = "STATUS_APPLIED"
	EventMonsterStatusEffectExpired    = "STATUS_EXPIRED"
	EventMonsterStatusEffectCancelled  = "STATUS_CANCELLED"
	EventMonsterStatusDamageReflected  = "DAMAGE_REFLECTED"
	EventMonsterStatusFriendlyDrop     = "FRIENDLY_DROP"
	EventMonsterStatusAggroChanged     = "AGGRO_CHANGED"
	EventMonsterStatusNextSkillDecided = "NEXT_SKILL_DECIDED"
	EventMonsterStatusMpChanged        = "MP_CHANGED"

	MpChangeReasonMpEater = "MP_EATER"
)
```

(Locate the existing const block — preserve all existing entries; only the
last two lines are new. The block layout above shows the final form.)

- [ ] **Step 2: Add the body type**

Append the body struct near the other `statusEvent*Body` types in the same
file (find an existing one like `statusEventDamageReflectedBody` and add this
adjacent to it):

```go
type statusEventMpChangedBody struct {
	CharacterId    uint32 `json:"characterId"`
	SkillId        uint32 `json:"skillId"`
	Reason         string `json:"reason"`
	Amount         uint32 `json:"amount"`
	MonsterMpAfter uint32 `json:"monsterMpAfter"`
}
```

- [ ] **Step 3: Verify it compiles**

```bash
cd services/atlas-monsters/atlas.com/monsters && go build ./monster/...
```

Expected: no output (success).

- [ ] **Step 4: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/kafka.go
git commit -m "feat(atlas-monsters): add MP_CHANGED event type and MP_EATER reason"
```

---

## Task 7: Add `mpChangedStatusEventProvider` in atlas-monsters

Producer for the new status event, mirroring `damageReflectedEventProvider`.

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/producer.go`

- [ ] **Step 1: Append the provider**

Open `services/atlas-monsters/atlas.com/monsters/monster/producer.go` and add
this function near `damageReflectedEventProvider` (line 116):

```go
// mpChangedStatusEventProvider builds a MP_CHANGED status event for any
// monster MP mutation that the channel must react to. Reason
// disambiguates the source (e.g., MP_EATER) so future passives can share
// the channel without expanding the consumer surface. Amount is the
// actual amount drained (post-clamp); MonsterMpAfter is the monster's
// MP after the deduction.
func mpChangedStatusEventProvider(m Model, characterId uint32, skillId uint32, reason string, amount uint32) model.Provider[[]kafka.Message] {
	return statusEventProvider(m.Field(), m.UniqueId(), m.MonsterId(), EventMonsterStatusMpChanged, statusEventMpChangedBody{
		CharacterId:    characterId,
		SkillId:        skillId,
		Reason:         reason,
		Amount:         amount,
		MonsterMpAfter: m.Mp(),
	})
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd services/atlas-monsters/atlas.com/monsters && go build ./monster/...
```

Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/producer.go
git commit -m "feat(atlas-monsters): add mpChangedStatusEventProvider"
```

---

## Task 8: Add `Processor.DrainMp` in atlas-monsters

The mutation that drains the monster's MP and emits `MP_CHANGED`. This is the
authoritative end of the contract.

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/processor.go`
- Test: `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go` (or new `drain_mp_test.go` in the same package)

- [ ] **Step 1: Write the failing happy-path test**

Pick one of:
(a) append to the existing `processor_test.go`
(b) create `services/atlas-monsters/atlas.com/monsters/monster/drain_mp_test.go`

If (b), start the file with the same package declaration and imports the
existing `processor_test.go` uses. Read `processor_test.go` once and reuse the
helper that constructs a `ProcessorImpl` with an in-memory registry and an
intercepting `emit` (the existing tests reference `setupProcessor` /
`fakeEmitter`-style helpers — discover the exact name when reading the file
and reuse it).

The new test, regardless of location:

```go
func TestDrainMp_HappyPath_EmitsMpChanged(t *testing.T) {
	tm, p, captured := newTestProcessor(t) // helper from existing tests; reuse
	m := seedMonster(t, tm, 9300000 /* non-boss template */, 1000 /* MaxMp */, 1000 /* Mp */)

	if err := p.DrainMp(m.UniqueId(), 42 /* characterId */, 2300000 /* ClericMpEaterId */, 100); err != nil {
		t.Fatalf("DrainMp returned error: %v", err)
	}

	got, err := GetMonsterRegistry().GetMonster(tm, m.UniqueId())
	if err != nil {
		t.Fatalf("registry lookup: %v", err)
	}
	if got.Mp() != 900 {
		t.Fatalf("Mp() = %d; want 900", got.Mp())
	}

	if len(captured.events) != 1 {
		t.Fatalf("captured %d events; want 1", len(captured.events))
	}
	ev := captured.events[0]
	if ev.Type != EventMonsterStatusMpChanged {
		t.Fatalf("event type = %q; want MP_CHANGED", ev.Type)
	}
	body := ev.Body.(statusEventMpChangedBody)
	if body.Reason != MpChangeReasonMpEater {
		t.Fatalf("Reason = %q; want MP_EATER", body.Reason)
	}
	if body.Amount != 100 {
		t.Fatalf("Amount = %d; want 100", body.Amount)
	}
	if body.MonsterMpAfter != 900 {
		t.Fatalf("MonsterMpAfter = %d; want 900", body.MonsterMpAfter)
	}
	if body.CharacterId != 42 {
		t.Fatalf("CharacterId = %d; want 42", body.CharacterId)
	}
	if body.SkillId != 2300000 {
		t.Fatalf("SkillId = %d; want 2300000", body.SkillId)
	}
}
```

If `newTestProcessor` / `seedMonster` / `captured` are not the actual
helper/field names in this repo, the executing agent should:
1. Read `processor_test.go` start-to-end.
2. Find the existing `Damage` test or `UseSkill` test that exercises emit.
3. Reuse those exact helpers.
4. Adjust the test above to match the helper signatures.

The shape of the assertions stays the same; only the construction call changes.

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run TestDrainMp_HappyPath_EmitsMpChanged
```

Expected: **FAIL** with `p.DrainMp undefined` (or compile error).

- [ ] **Step 3: Add `DrainMp` to the `Processor` interface**

Open `services/atlas-monsters/atlas.com/monsters/monster/processor.go`. In the
`Processor` interface (lines 25-55), append after `RepickAndEmit`:

```go
type Processor interface {
	// ... existing entries ...
	RepickAndEmit(uniqueId uint32, reason RepickReason) error
	DrainMp(uniqueId uint32, characterId uint32, skillId uint32, requestedAmount uint32) error
}
```

- [ ] **Step 4: Implement `DrainMp` on `ProcessorImpl`**

Append at the bottom of the same file (after `attackerInField`):

```go
// DrainMp deducts MP from a monster as the result of a player passive
// (currently MP Eater). It re-checks Boss / MaxMp / current Mp guards
// against atlas-monsters' authoritative state, clamps the deduction at
// zero via DeductMp, and emits a MP_CHANGED status event with the
// supplied reason and the actual amount removed. Bosses, dry monsters,
// and missing/dead monsters short-circuit to nil with no event so the
// channel never plays a misleading visual.
func (p *ProcessorImpl) DrainMp(uniqueId uint32, characterId uint32, skillId uint32, requestedAmount uint32) error {
	m, err := GetMonsterRegistry().GetMonster(p.t, uniqueId)
	if err != nil {
		p.l.WithError(err).Debugf("DRAIN_MP: monster [%d] not found.", uniqueId)
		return nil
	}
	if !m.Alive() {
		return nil
	}

	if m.MaxMp() == 0 || m.Mp() == 0 || requestedAmount == 0 {
		return nil
	}

	if ma, infoErr := information.GetById(p.l)(p.ctx)(m.MonsterId()); infoErr == nil && ma.Boss() {
		return nil
	}

	preMp := m.Mp()
	post, err := GetMonsterRegistry().DeductMp(p.t, uniqueId, requestedAmount)
	if err != nil {
		return err
	}

	actual := preMp - post.Mp()
	if actual == 0 {
		return nil
	}

	return p.emit(EnvEventTopicMonsterStatus, mpChangedStatusEventProvider(post, characterId, skillId, MpChangeReasonMpEater, actual))
}
```

- [ ] **Step 5: Run the happy-path test to verify it passes**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run TestDrainMp_HappyPath_EmitsMpChanged -v
```

Expected: **PASS**.

- [ ] **Step 6: Add the negative-case tests**

Append to the same test file:

```go
func TestDrainMp_ClampsAtZero(t *testing.T) {
	tm, p, captured := newTestProcessor(t)
	m := seedMonster(t, tm, 9300000, 100, 50)

	if err := p.DrainMp(m.UniqueId(), 42, 2300000, 200); err != nil {
		t.Fatalf("DrainMp: %v", err)
	}

	got, _ := GetMonsterRegistry().GetMonster(tm, m.UniqueId())
	if got.Mp() != 0 {
		t.Fatalf("Mp() = %d; want 0", got.Mp())
	}
	if len(captured.events) != 1 {
		t.Fatalf("captured %d events; want 1", len(captured.events))
	}
	body := captured.events[0].Body.(statusEventMpChangedBody)
	if body.Amount != 50 {
		t.Fatalf("Amount = %d; want 50 (current MP at entry)", body.Amount)
	}
}

func TestDrainMp_SkipsZeroMaxMp(t *testing.T) {
	tm, p, captured := newTestProcessor(t)
	m := seedMonster(t, tm, 9300000, 0, 0)

	if err := p.DrainMp(m.UniqueId(), 42, 2300000, 50); err != nil {
		t.Fatalf("DrainMp: %v", err)
	}
	if len(captured.events) != 0 {
		t.Fatalf("captured %d events; want 0", len(captured.events))
	}
}

func TestDrainMp_SkipsZeroCurrentMp(t *testing.T) {
	tm, p, captured := newTestProcessor(t)
	m := seedMonster(t, tm, 9300000, 1000, 0)

	if err := p.DrainMp(m.UniqueId(), 42, 2300000, 50); err != nil {
		t.Fatalf("DrainMp: %v", err)
	}
	if len(captured.events) != 0 {
		t.Fatalf("captured %d events; want 0", len(captured.events))
	}
}

func TestDrainMp_SkipsZeroRequest(t *testing.T) {
	tm, p, captured := newTestProcessor(t)
	m := seedMonster(t, tm, 9300000, 1000, 1000)

	if err := p.DrainMp(m.UniqueId(), 42, 2300000, 0); err != nil {
		t.Fatalf("DrainMp: %v", err)
	}
	if len(captured.events) != 0 {
		t.Fatalf("captured %d events; want 0", len(captured.events))
	}
	got, _ := GetMonsterRegistry().GetMonster(tm, m.UniqueId())
	if got.Mp() != 1000 {
		t.Fatalf("Mp() = %d; want 1000", got.Mp())
	}
}

func TestDrainMp_MissingMonster(t *testing.T) {
	tm, p, captured := newTestProcessor(t)
	_ = tm
	if err := p.DrainMp(99999 /* unseeded */, 42, 2300000, 50); err != nil {
		t.Fatalf("DrainMp: %v", err)
	}
	if len(captured.events) != 0 {
		t.Fatalf("captured %d events; want 0", len(captured.events))
	}
}
```

For the boss-skip test, check whether the existing test infrastructure already
seeds an `information.Model` lookup or stubs `information.GetById` (search for
`testInformationLookup` in the package). If yes, set the Boss flag through
that seam:

```go
func TestDrainMp_SkipsBoss(t *testing.T) {
	tm, p, captured := newTestProcessor(t)
	m := seedMonster(t, tm, 8800000 /* boss template */, 1000, 1000)

	// Stub information.GetById to return Boss=true for this template.
	// Use whatever seam the existing tests use (e.g., testInformationLookup
	// var or a fake set on the processor).
	stubBossLookup(t, m.MonsterId(), true)

	if err := p.DrainMp(m.UniqueId(), 42, 2300000, 100); err != nil {
		t.Fatalf("DrainMp: %v", err)
	}
	if len(captured.events) != 0 {
		t.Fatalf("captured %d events; want 0", len(captured.events))
	}
	got, _ := GetMonsterRegistry().GetMonster(tm, m.UniqueId())
	if got.Mp() != 1000 {
		t.Fatalf("Mp() = %d; want 1000", got.Mp())
	}
}
```

If no seam exists for stubbing `information.GetById` in tests (i.e., the
existing UseBasicAttack/UseSkill tests do not stub it), the executing agent
should add a tiny package-level test override variable mirroring the existing
`testInformationLookup` pattern (processor.go:64 already declares one for
UseBasicAttack — extend its use to `DrainMp`, or add a second one if it's
already locked to `UseBasicAttack`). Either way, document it inline in the
processor file with a short comment.

- [ ] **Step 7: Run the full new test set**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/ -run TestDrainMp -v
```

Expected: **PASS** for all `TestDrainMp_*` tests.

- [ ] **Step 8: Run the whole package to confirm no regression**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/...
```

Expected: **PASS**.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/processor.go services/atlas-monsters/atlas.com/monsters/monster/processor_test.go services/atlas-monsters/atlas.com/monsters/monster/drain_mp_test.go
git commit -m "feat(atlas-monsters): add DrainMp processor with boss/clamp guards"
```

(Drop the `drain_mp_test.go` from the `git add` if you appended to
`processor_test.go` instead.)

---

## Task 9: Register `handleDrainMpCommand` in atlas-monsters consumer

Wire the consumer entry point to `Processor.DrainMp`.

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go`

- [ ] **Step 1: Append the handler**

Open `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go`.
Append after `handleUseBasicAttackCommand` (around line 146):

```go
func handleDrainMpCommand(l logrus.FieldLogger, ctx context.Context, c command[drainMpCommandBody]) {
	if c.Type != CommandTypeDrainMp {
		return
	}

	p := monster.NewProcessor(l, ctx)
	if err := p.DrainMp(c.MonsterId, c.Body.CharacterId, c.Body.SkillId, c.Body.Amount); err != nil {
		l.WithError(err).Errorf("DRAIN_MP failed for monster [%d] character [%d].", c.MonsterId, c.Body.CharacterId)
	}
}
```

- [ ] **Step 2: Register the handler**

In `InitHandlers` (lines 27-67), inside the existing chain of registrations
on the `EnvCommandTopic` topic, add a new registration alongside
`handleUseBasicAttackCommand`. Pick any spot before the
`EnvCommandTopicMovement` lookup:

```go
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleUseBasicAttackCommand))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleDrainMpCommand))); err != nil {
			return err
		}
```

- [ ] **Step 3: Verify it compiles**

```bash
cd services/atlas-monsters/atlas.com/monsters && go build ./...
```

Expected: no output (success).

- [ ] **Step 4: Run kafka package tests**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./kafka/...
```

Expected: **PASS**.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go
git commit -m "feat(atlas-monsters): wire DRAIN_MP consumer"
```

---

## Task 10: Add `DrainMpCommandProvider` and `Processor.DrainMp` in atlas-channel

The producer side. Mirrors `Damage` / `DamageCommandProvider`.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/monster/producer.go`
- Modify: `services/atlas-channel/atlas.com/channel/monster/processor.go`

- [ ] **Step 1: Append the producer**

Open `services/atlas-channel/atlas.com/channel/monster/producer.go`. After
`DamageCommandProvider` at the bottom of the file, append:

```go
// DrainMpCommandProvider builds the DRAIN_MP command for atlas-monsters
// to deduct MP from a monster as the result of a player passive (e.g.,
// MP Eater). atlas-monsters re-checks all guards (Boss / MaxMp / Mp) and
// clamps the deduction; on a non-zero drain it emits MP_CHANGED back to
// the channel so the caster's MP is refunded and the visual is played.
func DrainMpCommandProvider(f field.Model, monsterId uint32, characterId uint32, skillId uint32, amount uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(monsterId))
	value := &monster2.Command[monster2.DrainMpCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		MonsterId: monsterId,
		Type:      monster2.CommandTypeDrainMp,
		Body: monster2.DrainMpCommandBody{
			CharacterId: characterId,
			SkillId:     skillId,
			Amount:      amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 2: Append the processor method**

Open `services/atlas-channel/atlas.com/channel/monster/processor.go`. Append
after the `CancelStatus` method:

```go
// DrainMp emits a DRAIN_MP command instructing atlas-monsters to deduct
// MP from a monster as the result of a player passive. The channel
// pre-screens cheap guards (MaxMp/Mp non-zero); atlas-monsters does the
// authoritative boss check and final clamp. The actual proc visual and
// caster MP refund are deferred to the MP_CHANGED return event.
func (p *Processor) DrainMp(f field.Model, monsterId uint32, characterId uint32, skillId uint32, amount uint32) error {
	p.l.Debugf("Draining MP from monster [%d] for character [%d] via skill [%d]. Amount [%d].", monsterId, characterId, skillId, amount)
	return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(DrainMpCommandProvider(f, monsterId, characterId, skillId, amount))
}
```

- [ ] **Step 3: Verify it compiles**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./monster/...
```

Expected: no output (success).

- [ ] **Step 4: Run monster package tests**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./monster/...
```

Expected: **PASS** (no test changes; just confirming nothing regressed).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/monster/producer.go services/atlas-channel/atlas.com/channel/monster/processor.go
git commit -m "feat(atlas-channel): emit DRAIN_MP command from monster processor"
```

---

## Task 11: Consume `MP_CHANGED` in atlas-channel — refund and visual

The channel reacts to the return event by refunding the caster's MP and
broadcasting the SKILL_SPECIAL effect.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go`

- [ ] **Step 1: Append the handler**

Open `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go`.
Add the imports needed if absent — `_map` (already imported), `session`
(already imported), `socketHandler "atlas-channel/socket/handler"`. Search for
existing `socketHandler` import name; if a different alias is used in this
file (likely none for now), pick `socketHandler` to match the heal package's
convention. If the alias collides with an existing import, use a unique
local alias like `effectsh` and substitute throughout the new handler.

Append after `handleStatusEventNextSkillDecided` (line 524) — before the
final closing brace of the file:

```go
func handleStatusEventMpChanged(sc server.Model, wp writer.Producer) message.Handler[monster2.StatusEvent[monster2.StatusEventMpChangedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monster2.StatusEvent[monster2.StatusEventMpChangedBody]) {
		if e.Type != monster2.EventStatusMpChanged {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		switch e.Body.Reason {
		case monster2.MpChangeReasonMpEater:
			f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
			if err := character.NewProcessor(l, ctx).ChangeMP(f, e.Body.CharacterId, int16(e.Body.Amount)); err != nil {
				l.WithError(err).Errorf("MP_CHANGED MP_EATER: ChangeMP failed for character [%d].", e.Body.CharacterId)
			}

			sp := session.NewProcessor(l, ctx)
			_ = sp.IfPresentByCharacterId(e.ChannelId)(
				e.Body.CharacterId,
				socketHandler.AnnounceSkillSpecial(l)(ctx)(wp)(e.Body.SkillId),
			)
			_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(
				f, e.Body.CharacterId,
				socketHandler.AnnounceForeignSkillSpecial(l)(ctx)(wp)(e.Body.CharacterId, e.Body.SkillId),
			)
		default:
			l.Debugf("MP_CHANGED: ignoring unknown reason [%s] for monster [%d].", e.Body.Reason, e.UniqueId)
		}
	}
}
```

If `socketHandler` is not yet imported in this file, add the import:

```go
import (
	// ... existing imports ...
	socketHandler "atlas-channel/socket/handler"
)
```

If `session.NewProcessor` is not imported here yet, add `"atlas-channel/session"`.
(The `session` package is already used elsewhere in this file at the consumer
test seam; confirm via the existing imports at the top.)

- [ ] **Step 2: Register the handler**

In `InitHandlers` (lines 38-83), append the registration alongside the others
(immediately after `handleStatusEventNextSkillDecided`):

```go
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventNextSkillDecided(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMpChanged(sc, wp)))); err != nil {
					return err
				}
				return nil
```

- [ ] **Step 3: Verify it compiles**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./...
```

Expected: no output (success).

- [ ] **Step 4: Run the consumer package tests**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/monster/...
```

Expected: **PASS**.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go
git commit -m "feat(atlas-channel): handle MP_CHANGED MP_EATER — refund + visual"
```

---

## Task 12: Pure helpers — `resolveMpEaterSkillId`, `mpEaterShouldProc`, `mpEaterAbsorbAmount`

TDD-first. Three small pure functions; one test file.

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_mp_eater_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`

- [ ] **Step 1: Write the failing test file**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_mp_eater_test.go`:

```go
package handler

import (
	"math"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

func TestMpEaterShouldProc(t *testing.T) {
	cases := []struct {
		name string
		prop float64
		roll float64
		want bool
	}{
		{"prop 1.0 always true", 1.0, 0.99, true},
		{"prop 1.0 with zero roll", 1.0, 0.0, true},
		{"prop 0.5 roll under", 0.5, 0.49, true},
		{"prop 0.5 roll equal", 0.5, 0.50, false},
		{"prop 0.5 roll over", 0.5, 0.51, false},
		{"prop 0.0 never", 0.0, 0.0, false},
		{"negative prop never", -1.0, 0.0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mpEaterShouldProc(tc.prop, tc.roll); got != tc.want {
				t.Fatalf("mpEaterShouldProc(%v, %v) = %v; want %v", tc.prop, tc.roll, got, tc.want)
			}
		})
	}
}

func TestResolveMpEaterSkillId(t *testing.T) {
	cases := []struct {
		name      string
		jobId     job.Id
		wantId    skill.Id
		wantOk    bool
	}{
		{"Magician (200)", job.MagicianId, skill.Id(2000000), false},
		{"FPWizard (210)", job.FirePoisonWizardId, skill.FirePoisionWizardMpEaterId, true},
		{"FPMage (211)", job.FirePoisonMagicianId, skill.FirePoisionWizardMpEaterId, true},
		{"FPArchMage (212)", job.FirePoisonArchMagicianId, skill.FirePoisionWizardMpEaterId, true},
		{"ILWizard (220)", job.IceLightningWizardId, skill.IceLightningWizardMpEaterId, true},
		{"ILMage (221)", job.IceLightningMagicianId, skill.IceLightningWizardMpEaterId, true},
		{"ILArchMage (222)", job.IceLightningArchMagicianId, skill.IceLightningWizardMpEaterId, true},
		{"Cleric (230)", job.ClericId, skill.ClericMpEaterId, true},
		{"Priest (231)", job.PriestId, skill.ClericMpEaterId, true},
		{"Bishop (232)", job.BishopId, skill.ClericMpEaterId, true},
		{"Fighter (110) — no MP Eater", job.FighterId, skill.Id(1100000), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotId, gotOk := resolveMpEaterSkillId(tc.jobId)
			if gotId != tc.wantId {
				t.Fatalf("resolveMpEaterSkillId(%v) id = %v; want %v", tc.jobId, gotId, tc.wantId)
			}
			if gotOk != tc.wantOk {
				t.Fatalf("resolveMpEaterSkillId(%v) ok = %v; want %v", tc.jobId, gotOk, tc.wantOk)
			}
		})
	}
}

func TestMpEaterAbsorbAmount(t *testing.T) {
	cases := []struct {
		name  string
		maxMp uint32
		x     int16
		want  uint32
	}{
		{"normal", 1000, 10, 100},
		{"zero MaxMp", 0, 10, 0},
		{"zero X", 1000, 0, 0},
		{"negative X", 1000, -1, 0},
		{"large MaxMp does not overflow", math.MaxUint32, 10, math.MaxUint32 / 10},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mpEaterAbsorbAmount(tc.maxMp, tc.x); got != tc.want {
				t.Fatalf("mpEaterAbsorbAmount(%d, %d) = %d; want %d", tc.maxMp, tc.x, got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail to compile**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run "TestMpEater|TestResolveMpEater"
```

Expected: **FAIL** with `mpEaterShouldProc undefined`, `resolveMpEaterSkillId undefined`, `mpEaterAbsorbAmount undefined`.

- [ ] **Step 3: Implement the three helpers**

Open `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`.
Add an import for `job` if absent — check the existing imports for
`"github.com/Chronicle20/atlas/libs/atlas-constants/job"`. The `skill3` alias
for the constants package is already imported.

Append the three helpers near the top of the file, after
`attackKindFromAttackType` (line 74) and before `processAttack`:

```go
// resolveMpEaterSkillId derives the candidate MP Eater skill id from the
// caster's job using the Cosmic formula: (jobId - jobId%10) * 10000.
// Returns ok=false when the computed id is not a registered skill (e.g.,
// 1st-job Magician 200 → 2000000, which is not a real skill).
func resolveMpEaterSkillId(jobId job.Id) (skill3.Id, bool) {
	candidate := skill3.Id(uint32(uint16(jobId)-uint16(jobId)%10) * 10000)
	_, ok := skill3.Skills[candidate]
	return candidate, ok
}

// mpEaterShouldProc returns true when MP Eater should fire given the
// skill's prop and a single uniform roll in [0,1). Mirrors Cosmic's
// `prop == 1.0 || rand() < prop`. Defensive against negative props.
func mpEaterShouldProc(prop float64, roll float64) bool {
	if prop <= 0 {
		return false
	}
	return prop >= 1.0 || roll < prop
}

// mpEaterAbsorbAmount computes the requested drain from monster MaxMp
// and the skill's X (absorb percent). Returns 0 when MaxMp is 0 or X is
// non-positive. atlas-monsters re-clamps to the monster's current MP.
func mpEaterAbsorbAmount(maxMp uint32, x int16) uint32 {
	if maxMp == 0 || x <= 0 {
		return 0
	}
	return uint32(uint64(maxMp) * uint64(x) / 100)
}
```

If `job` is not yet imported in `character_attack_common.go`, add it to the
import block:

```go
import (
	// ... existing imports
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)
```

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run "TestMpEater|TestResolveMpEater" -v
```

Expected: **PASS** for all three test functions.

- [ ] **Step 5: Confirm `skill.Skills` is the correct symbol**

```bash
grep -n "var Skills " libs/atlas-constants/skill/constants.go
```

Expected: a hit at line 2358 (`var Skills = map[Id]Skill{`). If grep returns
nothing, the registry was renamed and you must adapt `resolveMpEaterSkillId`
to use the actual symbol; do not invent a name. Also re-check
`libs/atlas-constants/job/constants.go` for the `MagicianId`,
`FirePoisonWizardId`, etc. constants — the test depends on these being
exact integer values.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_mp_eater_test.go
git commit -m "feat(atlas-channel): add MP Eater pure helpers (resolve / proc / amount)"
```

---

## Task 13: `mpEaterTryProc` orchestrator + call site

The composition layer that joins the helpers, the snapshot fetch, and the
producer. Replaces the `// TODO Apply MPEater` comment.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`

- [ ] **Step 1: Append the orchestrator**

Open `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`.
Append after `mpEaterAbsorbAmount`:

```go
// mpEaterTryProc evaluates and (on success) emits MP Eater for one
// damaged monster. Called once per damaged monster after status apply.
// Errors are logged at Debugf/Errorf and swallowed — never abort the
// surrounding attack pipeline.
func mpEaterTryProc(
	l logrus.FieldLogger,
	ctx context.Context,
	mp *monster.Processor,
	c character.Model,
	monsterId uint32,
	field field.Model,
	characterId uint32,
) {
	eaterId, ok := resolveMpEaterSkillId(c.JobId())
	if !ok {
		return
	}

	var ownedLevel byte
	for _, owned := range c.Skills() {
		if owned.Id() == eaterId {
			ownedLevel = owned.Level()
			break
		}
	}
	if ownedLevel == 0 {
		return
	}

	eaterEffect, err := skill2.NewProcessor(l, ctx).GetEffect(uint32(eaterId), ownedLevel)
	if err != nil {
		l.WithError(err).Errorf("MP Eater: skill effect lookup failed for skill [%d] level [%d].", eaterId, ownedLevel)
		return
	}
	if eaterEffect.Prop() <= 0 {
		return
	}

	mon, err := mp.GetById(monsterId)
	if err != nil {
		l.WithError(err).Debugf("MP Eater: monster [%d] snapshot fetch failed.", monsterId)
		return
	}
	if mon.MaxMp() == 0 || mon.Mp() == 0 {
		return
	}

	if !mpEaterShouldProc(eaterEffect.Prop(), rand.Float64()) {
		return
	}

	amount := mpEaterAbsorbAmount(mon.MaxMp(), eaterEffect.X())
	if amount == 0 {
		return
	}

	l.Debugf("MP Eater proc: caster=[%d] skill=[%d] monster=[%d] amount=[%d] (monster MaxMp=%d Mp=%d).",
		characterId, eaterId, monsterId, amount, mon.MaxMp(), mon.Mp())

	if err := mp.DrainMp(field, monsterId, characterId, uint32(eaterId), amount); err != nil {
		l.WithError(err).Errorf("MP Eater: DRAIN_MP emit failed for monster [%d] caster [%d].", monsterId, characterId)
	}
}
```

Add the imports needed if not yet present (grep first):

```go
import (
	// ... existing
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)
```

(`field` is likely already imported transitively; check before adding to avoid
a duplicate import.)

- [ ] **Step 2: Wire the call site inside the per-damage-entry loop**

Locate the per-`di` loop in `processAttack` (starts at the `for _, di := range
ai.DamageInfo()` around line 153). Find the `ApplyStatus` block which today
ends near line 215 (closing `}` for the `if len(se.MonsterStatus()) > 0`
branch within the non-reflected path).

Immediately after that `ApplyStatus` block — but still inside the
non-reflected branch (i.e., after the `mp.Damage(...)` and `ApplyStatus(...)`
calls, before the closing `}` of the per-`di` iteration) — insert:

```go
						// MP Eater proc: per-monster, after status apply,
						// magic attacks only. Failures are swallowed so the
						// rest of the attack pipeline is unaffected.
						if ai.AttackType() == packetmodel.AttackTypeMagic && ai.SkillId() > 0 {
							mpEaterTryProc(l, ctx, mp, c, di.MonsterId(), s.Field(), s.CharacterId())
						}
```

The exact placement: after the closing `}` of the `if len(se.MonsterStatus())
> 0` block (currently line 215) and before the `}` that closes the
non-reflected branch / the per-`di` iteration. The orchestrator is *not*
inside the `len(se.MonsterStatus()) > 0` branch — it should fire on every
damaged monster, regardless of whether the skill applies a status.

If the executing agent is unsure where the per-`di` loop body actually ends
in the latest source, run:

```bash
grep -n "for _, di := range ai.DamageInfo()\|continue\|}$" services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go | head -40
```

and read the loop body in full before inserting. The intent: one
`mpEaterTryProc` call per non-reflected damaged monster, after both
`mp.Damage` and `ApplyStatus`.

- [ ] **Step 3: Remove the obsolete TODO**

In the same file, delete the line `// TODO Apply MPEater` (currently line 278,
in the trailing TODO block after the broadcast).

- [ ] **Step 4: Verify it compiles**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./...
```

Expected: no output (success).

- [ ] **Step 5: Run all socket/handler tests**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/...
```

Expected: **PASS** (existing handler tests + the three new MP Eater
helper tests).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go
git commit -m "feat(atlas-channel): wire MP Eater proc into processAttack"
```

---

## Task 14: Tick the TODO doc and confirm full builds

Final cleanup and verification.

**Files:**
- Modify: `docs/TODO.md`

- [ ] **Step 1: Tick off the TODO**

Open `docs/TODO.md`. Find the line `- [ ] Apply MPEater` (PRD references it at
line 90). Change it to `- [x] Apply MPEater`.

- [ ] **Step 2: Confirm no `Apply MPEater` TODO remains in code**

```bash
grep -RIn "TODO Apply MPEater\|TODO Apply MP[Ee]ater\|MPEater" services/atlas-channel/atlas.com/channel/
```

Expected: no hits in `socket/handler/character_attack_common.go`. (Hits in
helper / test code referring to `mpEater*` are fine.)

- [ ] **Step 3: Build atlas-channel**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./...
```

Expected: success.

- [ ] **Step 4: Test atlas-channel**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./...
```

Expected: **PASS** for all packages.

- [ ] **Step 5: Build atlas-monsters**

```bash
cd services/atlas-monsters/atlas.com/monsters && go build ./...
```

Expected: success.

- [ ] **Step 6: Test atlas-monsters**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./...
```

Expected: **PASS** for all packages.

- [ ] **Step 7: Commit**

```bash
git add docs/TODO.md
git commit -m "docs: mark Apply MPEater as done in TODO"
```

---

## Out-of-scope reminders

The following are explicitly NOT part of this task. Do not add them, even if
the surrounding code seems to invite it:

- Combo Drain, Pick Pocket, Energy Drain, Vampire, Mortal Blow, Hamstring,
  Slow, Blind, Paladin charges, etc. — separate tasks. The
  `AnnounceSkillSpecial` helper and `MP_CHANGED` event with `Reason` are
  intentionally reusable for them, but do not implement them here.
- Restructuring Heal's dual-packet architecture.
- New effect-data fields beyond `Prop` and `X` (already present).
- Cooldown / per-mob exclusion / anti-farm rate limiting beyond Cosmic's boss
  skip.
- Atlas UI changes.
- New metrics for proc rate.

---

## Acceptance reference

The PRD's §10 acceptance criteria map onto these tasks:

| Acceptance bullet | Covered by |
|---|---|
| Bishop heals Wraiths and procs MP Eater | Task 13 (call site fires on `AttackTypeMagic`; Heal damage half routes through `processAttack`) |
| FP/IL/Cleric magicians proc MP Eater on normal magic attacks | Task 13 |
| Bosses do not receive drain or visual | Task 8 (atlas-monsters boss check) |
| `MaxMp == 0` skip | Task 8 + Task 13 (defense in depth) |
| Current `Mp == 0` skip | Task 8 + Task 13 |
| Multi-line magic skill (Ice Strike) proc once per monster | Task 13 (call site is once per `di`, and `di` collapses to one entry per monster) |
| 1st-job Magician (job 200) and non-magicians no-op | Task 12 (`resolveMpEaterSkillId` returns `ok=false` for jobs whose computed id is not in `skill.Skills`) |
| Unit tests for roll, drain math, boss, MaxMp/current-Mp skip, 1st-job no-op | Task 8, Task 12 |
| atlas-monsters DrainMp clamps at zero, ignores missing/dead monster | Task 8 |
| No regressions in Heal / magic-attack / reflect / status-apply | Task 14 (full `go test ./...` for both services) |
| `// TODO Apply MPEater` removed; `docs/TODO.md:90` checked off | Task 13 + Task 14 |
| Builds for atlas-channel and atlas-monsters succeed; tests pass | Task 14 |

---

## Self-review — completed by author

1. **Spec coverage.** Every functional requirement in PRD §4 maps to a task
   above (resolution → 12; trigger conditions → 13 + 8; chance roll → 12;
   drain calc → 12 + 8; monster mutation → 8; caster refund → 11; visual →
   3 + 11; ordering → 13). PRD §5 (new Kafka command + body) → 4 + 5 + 7.
   §7 service impact items each map to specific tasks. §8 NFRs covered (no
   new locks, tenant headers via existing envelope, debug logging in
   orchestrator and processor).
2. **Placeholder scan.** No "TBD", "implement later", "add error handling",
   or "similar to Task N" in any step body. Every step that produces code
   shows the code in full. The one place a step says "use whatever helper
   the existing tests use" (Task 8 Step 1) is a *test-construction* step
   that must adapt to the actual local helper names — the assertion
   structure is fully written and the executing agent has explicit
   instructions to read `processor_test.go` first.
3. **Type / symbol consistency.**
   - `mpEaterTryProc` signature in Task 13 matches the call site insertion in
     Step 2 of Task 13 (`l, ctx, mp, c, di.MonsterId(), s.Field(),
     s.CharacterId()`).
   - `mp` in the call site is `*monster.Processor`, matching the
     orchestrator's parameter type (`mp *monster.Processor`). The existing
     `processAttack` already binds `mp := monster.NewProcessor(l, ctx)` at
     the top of the magic branch, so the parameter type matches.
   - `Processor.DrainMp` parameter order is identical in atlas-channel
     (Task 10) and atlas-monsters (Task 8): `(uniqueId, characterId,
     skillId, amount)` with `field` only in the channel variant (the
     channel needs the field to populate the Kafka envelope).
   - `MpChangeReasonMpEater` is `"MP_EATER"` in both kafka packages
     (Task 4 channel-side, Task 6 atlas-monsters-side).
   - `EventStatusMpChanged` (channel) vs `EventMonsterStatusMpChanged`
     (atlas-monsters) — both encode the same wire string `"MP_CHANGED"`
     and that asymmetry already exists for every other event constant
     (e.g., `EventStatusDamaged` vs `EventMonsterStatusDamaged`), so this is
     consistent with existing convention.
   - `skill.Skills` (not `skill.Registry`) used in helper Task 12 and called
     out explicitly so the executing agent does not write the wrong name
     from the design doc.
4. **Single-deviation flag.** Task 12 deliberately diverges from the design
   doc's literal text on two minor points: (a) `skill.Registry` →
   `skill.Skills` (the actual symbol), (b) defensive guard `prop <= 0`
   added to `mpEaterShouldProc` (the design's pseudocode lacked it; the
   PRD's "negative prop" defensive case at Task 12 Step 1 demands it). Both
   deviations are documented inline.

Plan is internally consistent and aligned with the design doc.
