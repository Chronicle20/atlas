# Effective Stats Character Integration - Context

**Last Updated:** 2026-02-03

---

## Key Files

### atlas-effective-stats Service

| File | Purpose |
|------|---------|
| `services/atlas-effective-stats/atlas.com/effective-stats/character/processor.go` | Core processor interface - `GetEffectiveStats()`, `SetBaseStats()`, bonus management |
| `services/atlas-effective-stats/atlas.com/effective-stats/character/registry.go` | In-memory singleton storing character effective stats models |
| `services/atlas-effective-stats/atlas.com/effective-stats/character/model.go` | Character model with base stats, bonuses, computed values |
| `services/atlas-effective-stats/atlas.com/effective-stats/stat/model.go` | `Computed` struct, `Bonus` struct, stat `Type` constants |
| `services/atlas-effective-stats/atlas.com/effective-stats/character/resource.go` | REST endpoint handler for GET effective stats |
| `services/atlas-effective-stats/atlas.com/effective-stats/kafka/consumer/buff/consumer.go` | Handles buff applied/expired events |
| `services/atlas-effective-stats/atlas.com/effective-stats/kafka/consumer/asset/consumer.go` | Handles equipment equip/unequip events |
| `services/atlas-effective-stats/atlas.com/effective-stats/kafka/message/buff/kafka.go` | Kafka message types for buff events |

### atlas-character Service

| File | Purpose |
|------|---------|
| `services/atlas-character/atlas.com/character/character/processor.go` | Core processor with `ChangeHP()`, `ChangeMP()`, level processing |
| `services/atlas-character/atlas.com/character/character/producer.go` | Kafka event producers for character status events |
| `services/atlas-character/atlas.com/character/kafka/message/character/kafka.go` | Kafka message types for character commands/events |
| `services/atlas-character/atlas.com/character/kafka/consumer/consumer.go` | Consumer initialization |

---

## Critical Code Locations

### TODOs to Address

**processor.go:976** - MP Growth needs effective Intelligence:
```go
// TODO this needs to incorporate computed total intelligence (buffs, weapons, etc)
resMax += uint16(math.Ceil(float64(c.Intelligence()) / 10))
```

**processor.go:1009** - ChangeHP needs effective MaxHP:
```go
// TODO consider effective (temporary) Max HP.
adjusted = enforceBounds(amount, c.HP(), c.MaxHP(), 0)
```

**processor.go:1081** - ChangeMP needs effective MaxMP:
```go
// TODO consider effective (temporary) Max MP.
adjusted := enforceBounds(amount, c.MP(), c.MaxMP(), 0)
```

### Effective Stats Registry Operations

**Add Bonus (registry.go:94-109):**
```go
func (r *Registry) AddBonus(t tenant.Model, worldId, channelId byte, characterId uint32, b stat.Bonus) Model {
    // ... gets/creates model
    m = m.WithBonus(b).Recompute()
    cm[characterId] = m
    return m
}
```

**Remove Bonus (registry.go:129-144):**
```go
func (r *Registry) RemoveBonusesBySource(t tenant.Model, characterId uint32, source string) (Model, error) {
    // ... gets model
    m = m.WithoutBonusesBySource(source).Recompute()
    cm[characterId] = m
    return m, nil
}
```

### Buff Event Flow

**Buff Expired Consumer (buff/consumer.go:70-83):**
```go
func handleBuffExpired(l logrus.FieldLogger, ctx context.Context, e buff.StatusEvent[buff.ExpiredStatusEventBody]) {
    if e.Type != buff.EventStatusTypeBuffExpired {
        return
    }
    p := character.NewProcessor(l, ctx)
    if err := p.RemoveBuffBonuses(e.CharacterId, e.Body.SourceId); err != nil {
        // error handling
    }
}
```

---

## Event Definitions

### Existing Events

**Character Status Event (kafka.go:175-205):**
```go
const EnvEventTopicCharacterStatus = "EVENT_TOPIC_CHARACTER_STATUS"
const StatusEventTypeStatChanged = "STAT_CHANGED"

type StatusEventStatChangedBody struct {
    ChannelId       channel.Id `json:"channelId"`
    ExclRequestSent bool       `json:"exclRequestSent"`
    Updates         []string   `json:"updates"`  // Only stat names, no values!
}
```

**Buff Status Event (buff/kafka.go):**
```go
const EnvEventStatusTopic = "EVENT_TOPIC_CHARACTER_BUFF_STATUS"
const EventStatusTypeBuffApplied = "APPLIED"
const EventStatusTypeBuffExpired = "EXPIRED"
```

### Proposed New Commands

**Clamp HP/MP Commands (added to existing COMMAND_TOPIC_CHARACTER):**
```go
const CommandClampHP = "CLAMP_HP"
const CommandClampMP = "CLAMP_MP"

type ClampHPBody struct {
    ChannelId channel.Id `json:"channelId"`
    MaxValue  uint16     `json:"maxValue"`
}

type ClampMPBody struct {
    ChannelId channel.Id `json:"channelId"`
    MaxValue  uint16     `json:"maxValue"`
}
```

These commands instruct character service to clamp current HP/MP to the provided max value:
- If current > maxValue, set current = maxValue
- If current <= maxValue, no-op

---

## Stat Types

**From stat/model.go:**
```go
const (
    TypeStrength      Type = "strength"
    TypeDexterity     Type = "dexterity"
    TypeLuck          Type = "luck"
    TypeIntelligence  Type = "intelligence"
    TypeMaxHP         Type = "max_hp"
    TypeMaxMP         Type = "max_mp"
    TypeWeaponAttack  Type = "weapon_attack"
    TypeWeaponDefense Type = "weapon_defense"
    TypeMagicAttack   Type = "magic_attack"
    TypeMagicDefense  Type = "magic_defense"
    TypeAccuracy      Type = "accuracy"
    TypeAvoidability  Type = "avoidability"
    TypeSpeed         Type = "speed"
    TypeJump          Type = "jump"
)
```

---

## Decisions Made

| Decision | Rationale | Date |
|----------|-----------|------|
| Event-driven clamping over polling | Reactive, real-time, no REST overhead on hot path | 2026-02-03 |
| Publish only MAX_DECREASED, not all changes | Minimal scope; can expand later | 2026-02-03 |
| Hybrid approach (events + REST) | REST acceptable for low-frequency level-up; events for real-time clamping | 2026-02-03 |

---

## Dependencies

### Go Module Dependencies

**atlas-character will need:**
- HTTP client for REST calls to effective-stats
- Handler for new `CommandClampHP` / `CommandClampMP` commands

**atlas-effective-stats will need:**
- Kafka producer for character commands
- Logic to detect MaxHP/MaxMP decreases

### Environment Variables

**New:**
- `EFFECTIVE_STATS_SERVICE_URL` - URL for atlas-effective-stats REST API

**Existing (used by effective-stats for commands):**
- `COMMAND_TOPIC_CHARACTER` - Kafka topic for character commands

### Kafka Topics

**Existing (consumed by effective-stats):**
- `EVENT_TOPIC_CHARACTER_BUFF_STATUS`
- `EVENT_TOPIC_ASSET_STATUS`
- `EVENT_TOPIC_SESSION_STATUS`

**Existing (produced by effective-stats, consumed by character):**
- `COMMAND_TOPIC_CHARACTER` - Now also used for `CommandClampHP` / `CommandClampMP`

---

## Related Issues

- `docs/TODO.md:10` - Character Attack Effects (27 unimplemented)
- `docs/TODO.md:14` - Damage Reduction Effects (10 unimplemented)
- `docs/TODO.md:119-124` - Character Service processor TODOs

---

## Testing Notes

### Buff Scenarios

**Hyper Body (Skill ID: 1301007):**
- Effect: +60% MaxHP, +60% MaxMP
- Duration: 300 seconds
- On apply: MaxHP/MaxMP increase
- On expire: MaxHP/MaxMP decrease → trigger clamping

### Equipment Scenarios

**HP Gear:**
- Any equipment with `Hp` stat bonus
- On equip: MaxHP increases
- On unequip: MaxHP decreases → trigger clamping if current > new max

### Edge Cases

1. Character dies while buff is active → buff expires → no clamping needed (HP already 0)
2. Multiple buffs expire simultaneously → multiple decreases → clamp to lowest
3. Equipment + buff both provide HP → remove one → may still be above base due to other
