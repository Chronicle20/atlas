# Multi-line damage batching — design

## Problem

For a single player attack, atlas-channel sends one Kafka `DAMAGE` command per
`(targeted monster, damage line)` tuple. atlas-monsters processes each
independently and, after each, emits either a `damaged` status event (monster
survived) or a `killed` status event (monster died from that line) — never
both.

The atlas-channel `damaged` handler writes the `MonsterHealth` (HP bar) packet;
the `killed` handler does not. Therefore, when the killing line of a multi-line
attack lands, no HP-bar update is sent for that line — the bar visually skips
the final drain and the monster jumps straight to its death animation.

Symptom (reported, reproduces on L7): a 2-line attack that kills shows only the
first line's HP-bar drain before death.

Source references:

- atlas-channel emits per-line: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:67-73`
- atlas-monsters dispatches damaged-or-killed: `services/atlas-monsters/atlas.com/monsters/monster/processor.go:230-308`
- atlas-channel writes HP bar only on damaged: `services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go:156`

## Goal

The HP bar drains through every damage line of a multi-line attack, including
the killing line. No race between lines is possible because the full attack is
applied as a single Kafka command.

## Non-goals

- The `MonsterDamage` popup bug at `consumer.go:188` (only the last
  `DamageEntries` entry is shown for `MonsterAttack` / `DamageOverTime`
  sources). Out of scope; tracked separately.
- Cross-monster atomicity within a single AoE attack. Each monster keeps its
  own Kafka command, since each has its own HP bar.
- Changes to the redis `applyDamageScript`. Damage lines are applied in a Go
  loop inside the consumer, one redis call per line, single goroutine.

## Approach

### Schema change (hard cut)

`services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go`:

```go
type damageCommandBody struct {
    CharacterId uint32   `json:"characterId"`
    Damages     []uint32 `json:"damages"`
    AttackType  byte     `json:"attackType"`
}
```

The previous `Damage uint32` field is removed (rename, not coexisting). atlas-channel
and atlas-monsters deploy together; in-flight messages with the old shape decode
as `Damages: nil` and the consumer treats them as no-op (logged).

The producer-side struct in atlas-channel
(`services/atlas-channel/atlas.com/channel/monster/kafka.go` — symmetric with the
consumer's `damageCommandBody`) gets the same field rename.

### Producer side (atlas-channel)

`services/atlas-channel/atlas.com/channel/monster/processor.go:43`:

```go
func (p *Processor) Damage(f field.Model, monsterId uint32, characterId uint32, damages []uint32, attackType byte) error
```

The corresponding `DamageCommandProvider` (`producer.go:84`) is updated to put
`damages` into the body.

`services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:67-73`
collapses the inner loop:

```go
for _, di := range ai.DamageInfo() {
    err := mp.Damage(s.Field(), di.MonsterId(), s.CharacterId(), di.Damages(), byte(ai.AttackType()))
    // ... existing status-effect application unchanged
}
```

Empty `di.Damages()` (which would currently be a no-op zero-iteration loop) is
skipped — no Kafka message is sent.

### Consumer side (atlas-monsters)

`services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go:66`:

```go
func handleDamageCommand(l logrus.FieldLogger, ctx context.Context, c command[damageCommandBody]) {
    if c.Type != CommandTypeDamage {
        return
    }
    if len(c.Body.Damages) == 0 {
        return
    }
    p := monster.NewProcessor(l, ctx)
    p.Damage(c.MonsterId, c.Body.CharacterId, c.Body.Damages, c.Body.AttackType)
}
```

`services/atlas-monsters/atlas.com/monsters/monster/processor.go:230` — `Damage`
takes a slice and applies lines in order:

```go
func (p *ProcessorImpl) Damage(id uint32, characterId uint32, damages []uint32, attackType byte) {
    m, err := GetMonsterRegistry().GetMonster(p.t, id)
    if err != nil { ... return }
    if !m.Alive() { ... return }

    // Reflect runs once per attack, not once per line.
    p.checkReflect(m, characterId, attackType)

    var isBoss bool
    var revives []uint32
    if mi, infoErr := information.GetById(p.l)(p.ctx)(m.MonsterId()); infoErr == nil {
        isBoss, revives = mi.Boss(), mi.Revives()
    }

    var last DamageSummary
    killed := false
    for _, d := range damages {
        s, err := GetMonsterRegistry().ApplyDamage(p.t, characterId, d, m.UniqueId())
        if err != nil { ... return }
        last = s
        if s.Killed {
            killed = true
            break // discard overkill
        }
    }

    // Always emit damaged so the channel writes the final HP-bar packet,
    // even on a killing attack.
    _ = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(
        damagedStatusEventProvider(last.Monster, characterId, characterId, isBoss,
            DamageSourceCharacterAttack, last.Monster.DamageSummary()))

    if killed {
        // Existing kill cleanup, unchanged from current code.
        GetCooldownRegistry().ClearCooldowns(p.ctx, p.t, id)
        GetDropTimerRegistry().Unregister(p.ctx, p.t, id)
        for _, se := range last.Monster.StatusEffects() {
            _ = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(
                statusEffectCancelledEventProvider(last.Monster, se))
        }
        _ = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(
            killedStatusEventProvider(last.Monster, characterId, isBoss, last.Monster.DamageSummary()))
        _, _ = GetMonsterRegistry().RemoveMonster(p.ctx, p.t, last.Monster.UniqueId())
        if len(revives) > 0 {
            p.spawnRevives(last.Monster, revives)
        }
        return
    }

    // Damage-leader re-control runs once after the full attack.
    if characterId != last.Monster.ControlCharacterId() {
        if last.Monster.DamageLeader() == characterId {
            m2, err := p.GetById(last.Monster.UniqueId())
            if err == nil {
                _ = p.StopControl(m2)
                _, _ = p.StartControl(m2.UniqueId(), characterId)
            }
        }
    }
}
```

The redis `applyDamageScript` is unchanged. Each iteration is one round-trip on
the consumer goroutine; ordering is preserved by virtue of a single sequential
loop.

### Channel-side event handling

`services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go:156`
(`handleStatusEventDamaged`) is unchanged — it already writes `MonsterHealth`
on every `damaged` event, which now also fires on the killing attack.

The killed handler is unchanged. The natural ordering is `damaged` then `killed`
on the same Kafka partition (keyed by `monsterId`), so the channel writes the
HP-bar drain to its terminal value, then plays the death animation.

## Behavior on edge cases

- **Empty `Damages`**: producer-side, the channel skips emitting; consumer-side,
  the handler returns early. Kafka topic stays clean.
- **Already-dead monster on arrival**: existing `!m.Alive()` early return at
  the top of `Damage` is preserved — no events emitted.
- **One-line attack**: `len(damages) == 1`, single iteration, identical
  semantics to today (one `damaged` or one `killed` event), with the new
  guarantee that a one-line killing blow now emits both `damaged` and
  `killed` (HP bar drains to 0% before death). This is a behavior change for
  one-line attacks too; it is intentional and matches the bug fix.
- **Killing line is not the last**: loop breaks after the killing line; later
  lines in the attack are dropped. `DamageSummary` reflects only what was
  actually applied. Damage-leader update is skipped (kill path runs instead).
- **Multi-monster AoE attack**: each `DamageInfo` produces its own Kafka
  command, partitioned by `monsterId`. Per-monster ordering is preserved by
  Kafka partitioning; cross-monster ordering is not coordinated, which is
  fine — each monster has its own HP bar.

## Testing

- **atlas-monsters unit** (`monster/processor_test.go` or analog):
  - Multi-line non-killing attack: emits exactly one `damaged` event; monster
    HP reflects sum of damages; `DamageEntries` length grows by `len(damages)`.
  - Multi-line killing attack (kill on last line): emits one `damaged` then
    one `killed`; monster removed from registry; cooldowns/drop timer cleared.
  - Multi-line killing attack (kill on middle line): later lines are not
    applied; `DamageEntries` reflects only applied lines; emits one `damaged`
    then one `killed`.
  - One-line killing attack: emits one `damaged` then one `killed` (this is a
    behavior change from today; assert it explicitly).
  - Empty `damages`: returns without redis or Kafka activity.
  - Already-dead monster: returns without redis or Kafka activity.
  - Damage-leader change applies once across the full attack, not per line.
- **atlas-monsters consumer unit** (`kafka/consumer/monster/...`): handler
  rejects empty `Damages`; calls `processor.Damage` with the slice.
- **atlas-channel producer unit**: `Damage(...)` produces a Kafka message whose
  body has `damages: [d1, d2]`.
- **atlas-channel handler unit**
  (`kafka/consumer/monster/consumer_test.go`): receiving a `damaged` event
  whose monster HP is 0 still writes a `MonsterHealth` packet (the killing-line
  fix). No assertion change for non-zero HP cases.
- **Manual smoke**: in-game, fire L7 at a monster you can two-shot; observe
  HP-bar drains visibly twice before death.

## Rollout

Hard cut. atlas-channel and atlas-monsters change in lockstep in a single PR.
No feature flag, no schema coexistence, no migration window. In-flight
messages on `COMMAND_TOPIC_MONSTER` during deploy that carry the old
`damage` field decode with `Damages: nil` and are dropped by the new
consumer's empty check (logged).
