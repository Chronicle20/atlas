# Design — Status Cure Consumables

## Status

Approved. Ready for `/plan-task`.

## Scope summary

Cure pots (`2050000`–`2050004`) are silently no-ops because `ApplyItemEffects` in atlas-consumables never reads the cure specs. This design wires those flags through to a new "cancel buffs whose stat changes intersect this set" API on atlas-buffs, dispatched async over the existing per-character buff command topic. The architecture is general enough to serve future "cure these stat types" callers (NPC clinic blessings, reactor cure-all, future Dispel/Heal-Self skills).

WZ data (`Item.wz/Consume/0205.img.xml`) was checked during design and corrected the PRD: the cure pots in scope are `2050000`–`2050004`; `2050005` has an empty spec and is out of scope. The `thaw` consumable spec is a freeze-resistance *buff* (appears on food/dish items only), not a cure flag — it is out of scope.

## Architecture decisions

Each decision below was confirmed during brainstorming. The rejected alternatives are recorded so future readers don't relitigate them.

### D1. Cancel granularity: whole buff, not per-stat-change

`CancelByStatTypes` removes any buff whose `Changes()` intersects the cure type set, dropping the entire buff from the registry and emitting one existing-shape `EXPIRED` event per cancelled buff.

- **Why:** mob debuffs in this codebase are stored as 1:1 standalone buffs keyed by `sourceId = monsterSkillId` (`atlas-monsters/.../monster/processor.go:67-85`), each with one stat-change. Whole-buff cancel is observably identical to per-stat-change cancel for every existing producer, and it preserves the immutable buff model and the existing `EXPIRED` event shape that atlas-channel already consumes.
- **Rejected:** "partial expiry" — mutate `Changes()` and emit a new event shape. Would require atlas-channel changes for zero today-observable benefit; would need to be revisited if a future producer ever packs a debuff and an unrelated stat-change into one source, but the right answer that day is to fix the producer, not complicate cure.

### D2. Async dispatch via `COMMAND_TOPIC_CHARACTER_BUFF`

atlas-consumables emits a new `CANCEL_BY_TYPES` command on the existing `COMMAND_TOPIC_CHARACTER_BUFF`. atlas-buffs's existing per-character Kafka key serializes it behind any earlier `APPLY` / `CANCEL` for the same character.

- **Why:** matches the existing producer-doesn't-know-buff-state pattern (consumables today fire `Cancel(sourceId)` without checking that the buff exists). Per-character partition key gives the ordering guarantee the cure semantics need without a sync round-trip.
- **Rejected:** synchronous REST. Adds new HTTP surface, breaks the established async pattern, blocks the consumable handler on a network call for a fire-and-forget operation.

### D3. Apply order in `ApplyItemEffects`: cure → HP/MP → status buffs

Cure dispatch happens *before* HP/MP recovery, which happens before status buff `Apply`.

- **Why:** cure and HP/MP go to different services (atlas-buffs and atlas-character respectively) on different topics, so emission order doesn't strictly serialize processing across services. But emitting cure first means a queued poison tick (also produced by atlas-buffs) lands behind the cancel — eliminating the small but observable race where a poison tick eats part of the heal between drink-time and cancel-commit-time.
- **Rejected:** keep current order, append cure at end. Same code complexity, leaves the heal-then-tick race in. Sync-wait-on-cure: ruled out by D2.

### D4. Always emit `CANCEL_BY_TYPES` when the consumable has cure flags; never query player state

If the consumable has *any* non-zero disease cure spec, atlas-consumables fires one `CANCEL_BY_TYPES` with the corresponding type list. If the player has no matching debuffs, atlas-buffs's `CancelByStatTypes` no-ops and emits zero `EXPIRED` events.

- **Why:** atlas-consumables is stateless wrt buff state and shouldn't acquire that knowledge for an optional optimization. One Kafka command per cure drink is cheap; the "no spurious traffic" success criterion in the PRD is about *downstream* `EXPIRED` events, not the upstream command.
- **Rejected:** query atlas-buffs first to skip emission when nothing to cancel. Re-introduces the sync coupling rejected in D2 for negligible Kafka volume savings.

### D5. Cure ignores Holy Shield

`CancelByStatTypes` does not call `hasImmunityBuff`. A character with `HOLY_SHIELD` who somehow has a debuff (e.g. shielded *after* getting hit) can still be cured.

- **Why:** Holy Shield is about gating *application* of new debuffs (`Apply` already checks it). Removal is a different operation and should not be gated by it — the player has chosen to use a cure pot, and silently no-opping it is counter-intuitive.
- **Rejected:** symmetric immunity check. Adds a confusing failure mode for no design benefit.

### D6. `thaw` is out of scope

`SpecTypeThaw` is not a cure flag. Per WZ data, `thaw` only appears on food/dish items in `Item.wz/Consume/0202.img.xml` and `0238.img.xml`, paired with a `time` duration — it is a freeze-resistance buff (mapping to the existing `TemporaryStatTypeThaw = "THAW"` constant). Wiring `thaw` up as a stat-up buff in `ApplyItemEffects` is a separate task and is not covered here.

## Components

### atlas-buffs

#### Registry: new method (`character/registry.go`)

```go
func (r *Registry) CancelByStatTypes(
    ctx context.Context,
    characterId uint32,
    typeSet map[string]bool,
) ([]buff.Model, error)
```

- One Get/Put cycle (mirrors the existing `CancelAll` shape).
- Returns the slice of cancelled buffs (caller emits `EXPIRED` events).
- Iterates `m.buffs`; for each buff, scans `b.Changes()` and includes the buff in the cancelled set if any change's `Type()` is in `typeSet`.
- Empty `typeSet` returns `(nil, nil)` without touching Redis.

#### Processor: new method (`character/processor.go`)

```go
type Processor interface {
    // ...existing...
    CancelByStatTypes(worldId world.Id, characterId uint32, types []string) error
}
```

```go
func (p *ProcessorImpl) CancelByStatTypes(worldId world.Id, characterId uint32, types []string) error {
    if len(types) == 0 {
        return nil
    }
    typeSet := make(map[string]bool, len(types))
    for _, t := range types {
        typeSet[t] = true
    }
    cancelled, err := GetRegistry().CancelByStatTypes(p.ctx, characterId, typeSet)
    if err != nil {
        return err
    }
    if len(cancelled) == 0 {
        return nil
    }
    return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
        for _, b := range cancelled {
            if err := buf.Put(character2.EnvEventStatusTopic,
                expiredStatusEventProvider(worldId, characterId, b.SourceId(), b.Level(), b.Duration(), b.Changes(), b.CreatedAt(), b.ExpiresAt())); err != nil {
                return err
            }
        }
        return nil
    })
}
```

No immunity check (per D5). Same `EXPIRED` event shape as today; atlas-channel needs no changes.

#### Kafka command (`kafka/message/character/kafka.go`)

```go
const CommandTypeCancelByTypes = "CANCEL_BY_TYPES"

type CancelByTypesCommandBody struct {
    Types []string `json:"types"`
}
```

Reuses the existing `Command[E]` envelope (`worldId`, `channelId`, `mapId`, `instance`, `characterId`, `type`, `body`) and the existing `EnvCommandTopic = "COMMAND_TOPIC_CHARACTER_BUFF"`.

#### Consumer handler (`kafka/consumer/character/consumer.go`)

New `handleCancelByTypes(l, ctx, c)` registered alongside the existing three (`handleApply`, `handleCancel`, `handleCancelAll`). Body:

```go
if c.Type != character2.CommandTypeCancelByTypes {
    return
}
if err := character.NewProcessor(l, ctx).CancelByStatTypes(c.WorldId, c.CharacterId, c.Body.Types); err != nil {
    l.WithError(err).Errorf("Unable to cancel buffs by types %v for character [%d].", c.Body.Types, c.CharacterId)
}
```

### atlas-consumables

#### Producer wrapper (`character/buff/`)

`kafka/message/character/buff/kafka.go` mirrors atlas-buffs:

```go
const CommandTypeCancelByTypes = "CANCEL_BY_TYPES"

type CancelByTypesCommandBody struct {
    Types []string `json:"types"`
}
```

`character/buff/producer.go` adds:

```go
func cancelByTypesCommandProvider(f field.Model, characterId uint32, types []string) model.Provider[[]kafka.Message]
```

`character/buff/processor.go` adds:

```go
func (p *Processor) CancelByTypes(f field.Model, characterId uint32, types []string) error
```

Single-message produce on `EnvCommandTopic`, mirroring the existing `Cancel` wrapper.

#### `ApplyItemEffects` refactor (`consumable/processor.go:72-156`)

Restructure so cure runs first:

```go
func ApplyItemEffects(l logrus.FieldLogger, ctx context.Context, c character.Model, f field.Model, ci consumable3.Model, characterId uint32, itemId item2.Id) {
    bp := buff.NewProcessor(l, ctx)
    cp := character.NewProcessor(l, ctx)

    // 1. Cure first.
    cureTypes := collectCureTypes(ci)
    if len(cureTypes) > 0 {
        _ = bp.CancelByTypes(f, characterId, cureTypes)
    }

    // 2. HP/MP recovery (existing logic, unchanged behavior).
    // 3. Status buff Apply at the end (existing logic, unchanged behavior).
}
```

`collectCureTypes(ci)` returns the list of stat-type strings for non-zero cure specs in this fixed mapping:

| Consumable spec | Stat type string |
|---|---|
| `SpecTypePoison` | `"POISON"` |
| `SpecTypeDarkness` | `"DARKNESS"` |
| `SpecTypeWeakness` | `"WEAKEN"` |
| `SpecTypeSeal` | `"SEAL"` |
| `SpecTypeCurse` | `"CURSE"` |

Stat type values match the existing `TemporaryStatType` constants in `libs/atlas-constants/character/temporary_stat.go` (raw strings used in the Kafka body to match the existing `StatChange.Type` convention; the registry compares against `Changes()[].Type()` which is the same string).

`SpecTypeThaw` is **not** in this mapping (per D6). The existing pass through the other specs (HP/MP, statups) is unchanged.

### atlas-data

No changes — the parser at `consumable/reader.go:135-139` already produces the cure flags this design consumes.

### atlas-channel

No changes — already consumes the existing `EXPIRED` events that this design produces.

## Data flow (cure path)

```
Player drinks cure pot
        │
        ▼
[atlas-consumables] ConsumeStandard → ApplyItemEffects
        │ collectCureTypes(ci) → ["POISON","DARKNESS",...]
        ▼
[atlas-consumables] bp.CancelByTypes(f, characterId, types)
        │
        │  CANCEL_BY_TYPES on COMMAND_TOPIC_CHARACTER_BUFF
        ▼
[atlas-buffs] handleCancelByTypes
        │
        ▼
[atlas-buffs] CancelByStatTypes (processor)
        │
        ▼
[atlas-buffs] Registry.CancelByStatTypes
        │   single Get → filter buffs → Put
        ▼
[atlas-buffs] for each cancelled buff:
        │   emit EXPIRED on EVENT_TOPIC_CHARACTER_BUFF_STATUS
        ▼
[atlas-channel] existing handler renders cancel packets to the v83 client
```

Subsequent HP/MP recovery (atlas-character) and status-up buff `Apply` (atlas-buffs) fire after the cure command in the same `ApplyItemEffects` invocation. Per-character partition key on the buff topic ensures any queued poison tick lands behind the cancel.

## Error handling

- `CancelByStatTypes` with empty types returns `nil` without Redis I/O.
- `CancelByStatTypes` against a character with no buffs (registry `ErrNotFound` from underlying tenant registry) returns `(nil, nil)`; processor emits no events.
- Producer-side `CancelByTypes` failures in atlas-consumables are logged but do not block the rest of `ApplyItemEffects` (HP/MP recovery still runs). This matches the existing fire-and-forget treatment of `bp.Apply` errors via the leading `_ =`.

## Testing strategy

### atlas-buffs

**Registry (`character/registry_test.go`):**
- `CancelByStatTypes` with empty `typeSet` → `(nil, nil)`, registry untouched.
- No matching buffs (e.g. character has only `HOLY_SYMBOL`, request `["POISON"]`) → empty slice, registry untouched.
- Single match → returns one buff; non-matching buffs preserved in the map.
- Multi-match across distinct sources (one POISON buff, one CURSE buff, request `["POISON","CURSE"]`) → both returned, both removed.
- Buff whose `Changes()` contains a non-matching stat type is preserved (boundary).

**Processor (`character/processor_test.go`):**
- Emits one `EXPIRED` per cancelled buff with the expected fields (`SourceId`, `Changes`, etc.).
- Holy Shield active + matching debuff → still cancelled (D5 guard).
- No matches → no `EXPIRED` events emitted.

**Consumer (`kafka/consumer/character/consumer_test.go`):**
- `handleCancelByTypes` ignores commands of other types.
- `handleCancelByTypes` calls processor with `(WorldId, CharacterId, Body.Types)`.

### atlas-consumables

**Processor (`consumable/processor_test.go` or new test file):**
- Each single-effect pot (`2050000`–`2050003`) drives `bp.CancelByTypes` with the expected single-type or two-type list.
- All Cure (`2050004`) drives `bp.CancelByTypes` with all five types in the expected order.
- A non-cure consumable (e.g. white potion `2000000`) drives no `CancelByTypes`.
- A cure pot with `HPRecovery` calls `CancelByTypes` *before* `ChangeHP` (verifiable via call-order recording on a stub buff/character processor pair, or via emission order on a recording producer).

## Documentation updates

- `services/atlas-buffs/docs/domain.md`: add `CancelByStatTypes` to the processor surface.
- `services/atlas-buffs/docs/kafka.md`: add `CANCEL_BY_TYPES` command body and consumer handler.
- `services/atlas-consumables/docs/domain.md`: document the cure contract — non-zero disease cure specs trigger one `CancelByTypes` dispatch, ordered before HP/MP recovery and status buffs.

## Out of scope (re-confirmed)

- `thaw` buff wiring (D6).
- Mob `Dispel` skill (`127`) — already handled separately.
- Reactor cure-all actions — tracked in `docs/TODO.md`.
- Holy Shield application semantics — already handled by `Apply`'s immunity check.
- UI / packet changes in atlas-channel — already correct.

## References

- PRD: `docs/tasks/task-051-status-cure-consumables/prd.md`.
- WZ check: `tmp/ec876921-c363-4cc6-9c51-5bb8d57f9553/GMS/83.1/Item.wz/Consume/0205.img.xml` (cure pots), `0202.img.xml` and `0238.img.xml` (thaw buff items).
- Existing buff cancel path: `services/atlas-buffs/atlas.com/buffs/character/processor.go:57-80`.
- Existing buff command topic: `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go:13-17`.
- Disease stat type whitelist: `services/atlas-buffs/atlas.com/buffs/character/immunity.go:7-11`.
- `ApplyItemEffects` (the gap): `services/atlas-consumables/atlas.com/consumables/consumable/processor.go:72-156`.
- atlas-consumables buff producer (pattern to mirror): `services/atlas-consumables/atlas.com/consumables/character/buff/{processor.go,producer.go}`.
