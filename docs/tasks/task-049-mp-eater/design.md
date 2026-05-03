# MP Eater Passive — Design Document

Version: v1
Status: Approved
Created: 2026-05-03
PRD: `docs/tasks/task-049-mp-eater/prd.md`

---

## 1. Scope of this document

The PRD already settled what is being built (MP Eater for FP Wizard / IL Wizard / Cleric variants), why (Cosmic-parity passive triggered on every magic attack, surfacing for Heal-vs-undead in particular), the user-visible behavior, and acceptance criteria. This document captures the architectural decisions made on top of that — specifically the open questions enumerated in PRD §9 — and the corresponding component-level changes.

Decisions recorded here:

1. **Authority split** — atlas-monsters owns the boss check, MP clamp, and emits the result; atlas-channel emits a "try drain" command and reacts to a return event for visual + caster MP refund. (PRD §9.1)
2. **Amount computation** — atlas-channel pre-computes the requested amount from its `MaxMp` snapshot and the skill's `X`; atlas-monsters re-clamps to current MP and reports the actual amount drained. (PRD §9.2)
3. **Return event shape** — new `MP_CHANGED` status event on the existing monster-status topic, carrying a `Reason` field so future MP-affecting passives (e.g., Magic Guard refund, Drain MP) can reuse the channel. v1 only emits `Reason = MP_EATER`. (PRD §9.2)
4. **RNG seam** — pure helper takes the random `roll` as a parameter; production callers pass `rand.Float64()`. Mirrors the `snapshotVenomDamagePerTick` pattern already in the same file. (PRD §9.3)
5. **Visual broadcaster** — add dedicated `AnnounceSkillSpecial` / `AnnounceForeignSkillSpecial` helpers in `socket/handler/effects.go`, mirroring the existing `AnnounceSkillUse` pair. Reusable for the other passive TODOs in the same block. (PRD §9.4)
6. **Code placement** — keep MP Eater orchestration inline in `character_attack_common.go` for v1; the three pure helpers live as top-level package functions in the same file. Extraction to a dedicated `mp_eater.go` is deferred until additional passives in the same TODO block land and the file's growth becomes the active concern.

---

## 2. Architecture

### 2.1 End-to-end flow

```
atlas-channel                                          atlas-monsters
─────────────                                          ──────────────
processAttack
  └─ per damage entry (after damage + status apply):
      └─ mpEaterTryProc
          ├─ gate: AttackTypeMagic, SkillId > 0
          ├─ resolveMpEaterSkillId(jobId)
          ├─ skill registry + caster owns level > 0
          ├─ skill effect lookup → (prop, x)
          ├─ mp.GetById(monsterId) snapshot
          ├─ guard: MaxMp > 0, Mp > 0
          ├─ mpEaterShouldProc(prop, rand.Float64())
          ├─ amount = mpEaterAbsorbAmount(MaxMp, x)
          └─ DRAIN_MP command  ───────────────────►  consumer/monster
                                                       └─ handleDrainMpCommand
                                                           └─ Processor.DrainMp
                                                               ├─ skip if missing/dead
                                                               ├─ skip if Boss / MaxMp==0 / Mp==0
                                                               ├─ DeductMp (atomic, clamped at 0)
                                                               └─ emit MP_CHANGED
                                                                   { Reason: MP_EATER, Amount, MpAfter }
                                                                                            │
consumer/monster (channel)                                                                  │
  └─ handleMpChanged   ◄──────────────  EVENT_TOPIC_MONSTER_STATUS  ◄────────────────────────┘
      when Reason == MP_EATER:
        ├─ cp.ChangeMP(field, characterId, +int16(Amount))
        ├─ AnnounceSkillSpecial   (caster session)
        └─ AnnounceForeignSkillSpecial (other sessions in same map)
```

### 2.2 Why monsters-authoritative

- **Single source of truth.** atlas-monsters already owns `MaxMp`, current `Mp`, `Boss`, and the atomic `DeductMp` mutation. Pushing the boss check and clamp into the channel would require duplicating the `Boss` flag into `monster.RestModel` / `monster.Model` and accepting that channel snapshots can lag the registry.
- **Defense in depth.** The channel still pre-screens for `MaxMp == 0` / `Mp == 0` to avoid wasted Kafka traffic, but atlas-monsters re-checks those plus `Boss` so a stale channel snapshot cannot drive a drain.
- **Visual-on-confirmation.** The visual + caster MP refund only fire after atlas-monsters confirms a non-zero drain. This means bosses, dry monsters, and lost-race-to-kill drains do not produce a misleading "MP Eater proc" effect.
- **Cost.** One additional Kafka emit per successful proc (the return event). Procs are a small fraction of magic attacks, and the topic already carries `DAMAGED` for every attack hit, so the relative overhead is negligible.

The trade-off accepted: the proc visual lands one Kafka round-trip after the hit. The same delay already applies to `DAMAGE_REFLECTED` and is not perceptible at gameplay scale.

### 2.3 Why pre-compute amount on channel

Per PRD-required precedent: the existing `DAMAGE` command sends pre-computed per-line damages, and atlas-monsters applies them. Mirroring that for `DRAIN_MP` keeps the protocol shape consistent (atlas-monsters mutates HP/MP based on amounts the channel hands it; it does not know what skill `X` is). atlas-monsters's job is to clamp — not to interpret skill data.

Staleness window: `MaxMp` is essentially immutable mid-fight (no v83 monster has buffs that change `MaxMp`), so the channel's snapshot can be trusted for the multiplication. The only meaningful staleness is on `Mp` — which is exactly what the consumer-side clamp covers.

---

## 3. Component changes

### 3.1 `libs/atlas-constants` — no change

Skill IDs (`FirePoisionWizardMpEaterId`, `IceLightningWizardMpEaterId`, `ClericMpEaterId`) and job assignments are already present.

### 3.2 `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go`

Add accessors over existing fields (no new state):

```go
func (m Model) Prop() float64 { return m.prop }
func (m Model) X() int16      { return m.x }
```

### 3.3 `services/atlas-channel/atlas.com/channel/monster/` (`model.go`, `builder.go`, `rest.go`)

Extend the live snapshot with `MaxMp` (RestModel already carries `MaxMp uint32`):

- `Model`: add `maxMp uint32` field.
- `Model.MaxMp() uint32` accessor.
- `modelBuilder.SetMaxMp(uint32)`.
- `Extract` wires `maxMp` from `RestModel.MaxMp`.

### 3.4 `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`

- Drop the `// TODO Apply MPEater` comment at line 278.
- Inside the existing per-damage-entry loop, after the `ApplyStatus` block (currently around line 215), call a new orchestrator `mpEaterTryProc(...)` that owns the per-monster proc evaluation.
- Add three top-level helpers in the same file (testable, pure):

```go
// resolveMpEaterSkillId derives the candidate MP Eater skill id from the
// caster's job using the Cosmic formula. Returns ok=false when the
// computed id is not in the skill registry (e.g., 1st-job Magician
// computes to 2000000, which does not exist).
func resolveMpEaterSkillId(jobId job.Id) (skill.Id, bool) {
    candidate := skill.Id((uint16(jobId) - uint16(jobId)%10) * 10000)
    _, ok := skill.Registry[candidate]
    return candidate, ok
}

// mpEaterShouldProc returns true when MP Eater should fire given a
// skill prop and a single uniform [0, 1) roll. Mirrors Cosmic:
// prop == 1.0 || roll < prop.
func mpEaterShouldProc(prop float64, roll float64) bool {
    return prop >= 1.0 || roll < prop
}

// mpEaterAbsorbAmount computes the requested drain from monster MaxMp
// and the skill's X (absorb percent). Returns 0 when MaxMp is 0 or X is
// non-positive. Channel-side computation; atlas-monsters re-clamps to
// the monster's current MP.
func mpEaterAbsorbAmount(maxMp uint32, x int16) uint32 {
    if maxMp == 0 || x <= 0 {
        return 0
    }
    return uint32(int64(maxMp) * int64(x) / 100)
}
```

`mpEaterTryProc` is called per damage entry inside the existing loop. Pseudocode:

```go
// inside `for _, di := range ai.DamageInfo()` after the ApplyStatus block
if ai.AttackType() == packetmodel.AttackTypeMagic && ai.SkillId() > 0 && !reflected {
    mpEaterTryProc(l, ctx, mp, cp, c, di, se, s, wp)
}
```

`mpEaterTryProc` performs (in order, each returning early on fail):

1. `eaterId, ok := resolveMpEaterSkillId(c.JobId())` — return if `!ok`.
2. Find the caster's owned skill matching `eaterId`. Return if not owned or level == 0.
3. `eaterEffect, err := skill2.NewProcessor(l, ctx).GetEffect(uint32(eaterId), ownedLevel)` — return on err. Return if `eaterEffect.Prop() <= 0`.
4. `mon, err := mp.GetById(di.MonsterId())` — return on err. Return if `mon.MaxMp() == 0` or `mon.Mp() == 0`.
5. `if !mpEaterShouldProc(eaterEffect.Prop(), rand.Float64())` — return.
6. `amount := mpEaterAbsorbAmount(mon.MaxMp(), eaterEffect.X())` — return if `amount == 0`.
7. `_ = mp.DrainMp(s.Field(), di.MonsterId(), s.CharacterId(), uint32(eaterId), amount)` — log Errorf on failure, do not abort.

The orchestrator never aborts the attack; all errors logged and swallowed.

Note on snapshot reuse: the reflect block also calls `mp.GetById(di.MonsterId())` for magic attacks, but only when `mirror.GetReflect` returns a non-zero entry. Plumbing the snapshot from reflect to MP Eater is structurally awkward (reflect lives in a conditional branch with its own early-`continue`). For v1 we accept the second fetch on the rare proc path. If telemetry shows the duplicate fetch is hot, a small refactor that hoists the snapshot above the reflect/MP-Eater branches is straightforward.

### 3.5 `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go`

Append:

```go
const (
    CommandTypeDrainMp        = "DRAIN_MP"

    EventStatusMpChanged      = "MP_CHANGED"

    MpChangeReasonMpEater     = "MP_EATER"
)

type DrainMpCommandBody struct {
    CharacterId uint32 `json:"characterId"`
    SkillId     uint32 `json:"skillId"`
    Amount      uint32 `json:"amount"`
}

type StatusEventMpChangedBody struct {
    CharacterId    uint32 `json:"characterId"`
    SkillId        uint32 `json:"skillId"`
    Reason         string `json:"reason"`
    Amount         uint32 `json:"amount"`
    MonsterMpAfter uint32 `json:"monsterMpAfter"`
}
```

`Reason` is a string (not a typed enum) so atlas-monsters and atlas-channel evolve independently as new MP-source/sink reasons land.

### 3.6 `services/atlas-channel/atlas.com/channel/monster/processor.go` and `producer.go`

- Add `Processor.DrainMp(f field.Model, monsterId, characterId, skillId, amount uint32) error` mirroring the shape of `Damage`:

```go
func (p *Processor) DrainMp(f field.Model, monsterId uint32, characterId uint32, skillId uint32, amount uint32) error {
    p.l.Debugf("Draining MP from monster [%d] for character [%d] via skill [%d]. Amount [%d].", monsterId, characterId, skillId, amount)
    return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(DrainMpCommandProvider(f, monsterId, characterId, skillId, amount))
}
```

- Add `DrainMpCommandProvider` next to `DamageCommandProvider`.

### 3.7 `services/atlas-channel/atlas.com/channel/socket/handler/effects.go`

Add new helpers next to `AnnounceSkillUse` / `AnnounceForeignSkillUse`:

```go
func AnnounceSkillSpecial(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(skillId uint32) model2.Operator[session.Model] {
    return func(ctx context.Context) func(wp writer.Producer) func(skillId uint32) model2.Operator[session.Model] {
        return func(wp writer.Producer) func(skillId uint32) model2.Operator[session.Model] {
            return func(skillId uint32) model2.Operator[session.Model] {
                return session.Announce(l)(ctx)(wp)(charcb.CharacterEffectWriter)(charpkt.CharacterSkillSpecialEffectBody(skillId))
            }
        }
    }
}

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

### 3.8 Channel-side `MP_CHANGED` consumer

Add a new handler in the channel's existing monster-status event consumer (`services/atlas-channel/atlas.com/channel/kafka/consumer/monster/...`):

- Gate on `Type == EventStatusMpChanged`.
- Switch on `Body.Reason`. For `MP_EATER`:
  1. Resolve caster's session via `session.NewProcessor(l, ctx).IfPresentByCharacterId(channel.Id)(...)`.
  2. Build `field.Model` from the event envelope.
  3. `_ = cp.ChangeMP(field, characterId, +int16(amount))`. Errorf on failure, continue.
  4. `socketHandler.AnnounceSkillSpecial(l)(ctx)(wp)(skillId)` against the caster session.
  5. `_map.NewProcessor(l, ctx).ForOtherSessionsInMap(field, characterId, socketHandler.AnnounceForeignSkillSpecial(l)(ctx)(wp)(characterId, skillId))`.
- Unknown reasons: log `Debugf`, no-op (forward-compatibility for future reasons added by atlas-monsters).

The producer for `wp` and the existing consumer wiring already passes a `writer.Producer` into other monster-status handlers (e.g., the existing `DAMAGE_REFLECTED` consumer applies the reflected HP delta and emits client packets); the `MP_CHANGED` handler follows that same pattern.

### 3.9 atlas-monsters — `kafka/consumer/monster/kafka.go`

Add the matching wire constants and types:

```go
const (
    CommandTypeDrainMp     = "DRAIN_MP"
    EventStatusMpChanged   = "MP_CHANGED"
    MpChangeReasonMpEater  = "MP_EATER"
)

type drainMpCommandBody struct {
    CharacterId uint32 `json:"characterId"`
    SkillId     uint32 `json:"skillId"`
    Amount      uint32 `json:"amount"`
}

type statusEventMpChangedBody struct {
    CharacterId    uint32 `json:"characterId"`
    SkillId        uint32 `json:"skillId"`
    Reason         string `json:"reason"`
    Amount         uint32 `json:"amount"`
    MonsterMpAfter uint32 `json:"monsterMpAfter"`
}
```

### 3.10 atlas-monsters — `kafka/consumer/monster/consumer.go`

Add `handleDrainMpCommand` and register it in `InitHandlers`:

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

### 3.11 atlas-monsters — `monster/processor.go`

Add:

```go
func (p *ProcessorImpl) DrainMp(uniqueId, characterId, skillId, requestedAmount uint32) error {
    m, err := GetMonsterRegistry().GetMonster(p.t, uniqueId)
    if err != nil {
        // Missing monster: drop silently, consistent with DAMAGE.
        p.l.WithError(err).Debugf("DRAIN_MP: monster [%d] not found.", uniqueId)
        return nil
    }
    if !m.Alive() {
        return nil
    }

    // Defense-in-depth: re-check guards even though channel pre-screens.
    if m.MaxMp() == 0 || m.Mp() == 0 || requestedAmount == 0 {
        return nil
    }

    // Boss flag may live on monster information (atlas-data) rather than the
    // registry model. Fetch it the same way Damage does.
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

`actual` is computed from the local snapshot's `preMp` minus the post-deduct `Mp()`. Concurrent drains against the same monster are serialized by the registry's atomic update; the local subtraction reflects exactly what *this* call removed.

`mpChangedStatusEventProvider` is a small new producer in atlas-monsters mirroring `damagedStatusEventProvider`.

### 3.12 No `atlas-character` change

The existing `character.Processor.ChangeMP` is reused unchanged for the caster MP refund.

### 3.13 No `libs/atlas-packet` change

`CharacterSkillSpecialEffectBody` and `CharacterSkillSpecialEffectForeignBody` already exist.

### 3.14 No `atlas-data` change

The skill effect endpoint already returns `Prop` and `X`. Validation that the data is populated for skill ids `2100000` / `2200000` / `2300000` at all known levels is part of acceptance, not implementation.

---

## 4. Test plan

### 4.1 atlas-channel unit tests

Co-located with `character_attack_common.go` (extend or add `*_test.go` in the same package):

- `TestMpEaterShouldProc` — table-driven:
  - `prop=1.0, roll=anything → true`
  - `prop=0.5, roll=0.49 → true`
  - `prop=0.5, roll=0.50 → false`
  - `prop=0.0, roll=anything → false`
  - `prop=-1.0` (defensive) `→ false`
- `TestResolveMpEaterSkillId`:
  - `Magician(200) → 2000000, ok=false` (not in registry)
  - `FPWizard(210) → 2100000, ok=true`
  - `FPMage(211) → 2100000, ok=true`
  - `FPArchMage(212) → 2100000, ok=true`
  - `ILWizard(220) → 2200000, ok=true`
  - `Cleric(230) → 2300000, ok=true`
  - `Priest(231) → 2300000, ok=true`
  - `Bishop(232) → 2300000, ok=true`
  - `Fighter(110) → 1100000, ok=false`
- `TestMpEaterAbsorbAmount`:
  - `(MaxMp=1000, X=10) → 100`
  - `(MaxMp=0, X=10) → 0`
  - `(MaxMp=1000, X=0) → 0`
  - `(MaxMp=1000, X=-1) → 0`
  - Large MaxMp doesn't overflow (use a value near `math.MaxUint32`)

### 4.2 atlas-monsters unit tests

Co-located with the new processor method:

- `TestDrainMpClampsAtZero` — request > current MP → monster Mp = 0; emitted event Amount = current MP at entry.
- `TestDrainMpSkipsBoss` — boss monster: no event, Mp unchanged.
- `TestDrainMpSkipsZeroMaxMp` — MaxMp = 0: no event, Mp unchanged.
- `TestDrainMpSkipsZeroMp` — current Mp = 0: no event, Mp unchanged.
- `TestDrainMpZeroRequest` — requestedAmount = 0: no event, Mp unchanged.
- `TestDrainMpEmitsEventWithReason` — happy path: emits `MP_CHANGED` with `Reason = MP_EATER`, correct `Amount`, correct `MonsterMpAfter`.
- `TestDrainMpMissingMonster` — unknown monster id: returns nil, no event.
- `TestDrainMpDeadMonster` — monster already dead: no event, Mp unchanged.

### 4.3 No new integration tests required

The composition (channel orchestrator + Kafka command + atlas-monsters consumer + return event + channel handler) is exercised end-to-end in any in-game smoke test of a Bishop healing undead. Acceptance §10 covers the live verification. Per the existing test patterns, no synthetic Kafka integration tests are added for this kind of pipeline change.

---

## 5. Failure modes and observability

- **Snapshot fetch failure** — `mp.GetById` returns error → `Debugf` (proc just doesn't fire). Consistent with the existing reflect path.
- **Skill-effect lookup failure** — `Errorf`, no proc.
- **Kafka emit failure (DRAIN_MP)** — `Errorf`, no proc, no visual.
- **Atlas-monsters `DeductMp` failure** — `Errorf` in the consumer, no event emitted, channel silently does not refund. This is consistent with how `DAMAGE` handles registry mutation failures.
- **Channel-side `MP_CHANGED` consumer failures** — each sub-step (`ChangeMP`, visual broadcast) logs `Errorf` on failure, continues.
- **Logging** — proc emits `Debugf` matching the verbosity of `mp.Damage`. No new metrics for v1.

---

## 6. Multi-tenancy and security

- All Kafka commands and events use the existing tenant header envelope (`consumer.TenantHeaderParser` already wired on both services).
- atlas-monsters scopes its monster lookup and mutation by `tenant.Model` (existing pattern preserved).
- Drain amount is server-computed end-to-end. No client-trusted fields.
- Boss exclusion is enforced server-side in atlas-monsters. The channel pre-check is best-effort and does not constitute the security boundary.

---

## 7. Sequencing (for plan phase)

A natural decomposition into independently committable steps:

1. Add `Prop()` / `X()` accessors on `effect.Model`. Pure mechanical.
2. Add `MaxMp` to channel `monster.Model`/builder/rest. Pure mechanical.
3. Add `AnnounceSkillSpecial` / `AnnounceForeignSkillSpecial` helpers.
4. Add `DRAIN_MP` command + `MP_CHANGED` event constants and body types in both services' kafka message packages.
5. atlas-monsters: `DrainMp` processor method + consumer handler + event provider + tests.
6. atlas-channel: `Processor.DrainMp` + producer.
7. atlas-channel: `MP_CHANGED` consumer handler (refund + visual).
8. atlas-channel: pure helpers (`resolveMpEaterSkillId`, `mpEaterShouldProc`, `mpEaterAbsorbAmount`) + unit tests.
9. atlas-channel: `mpEaterTryProc` orchestrator + call site in `processAttack` (replacing the TODO).
10. Remove `// TODO Apply MPEater` from the source and check off `docs/TODO.md:90`.
11. Build + test both services.

The plan-phase output (`plan.md`) will refine ordering, dependencies, and TDD slicing.

---

## 8. Out of scope

- Other passive/attack-side TODOs in the same block (Combo Drain, Pick Pocket, Energy Drain, Vampire, Mortal Blow, Hamstring, Slow, Blind, Paladin charges). Each is a separate task; the `AnnounceSkillSpecial` helper and `MP_CHANGED` event with `Reason` are deliberately reusable for them.
- Restructuring Heal's dual-packet architecture.
- New effect-data fields, registries, caches, or shared state.
- Cooldown, per-mob exclusion, or anti-farm rate limiting beyond Cosmic's boss skip.
- Atlas UI changes.
- Telemetry / metrics for proc rate.
