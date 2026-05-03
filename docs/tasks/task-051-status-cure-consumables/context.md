# Context — Status Cure Consumables

Quick reference for engineers (or subagents) executing `plan.md`. Treat `prd.md` and `design.md` as authoritative; this is a navigation map.

## What we're building

Cure pots (`2050000`–`2050004`) are silently no-ops because `ApplyItemEffects` in atlas-consumables never reads the disease cure specs. We add:

1. A new `CancelByStatTypes(worldId, characterId, types)` API to atlas-buffs that drops any buff whose stat-changes intersect a given set of disease stat types, emitting one `EXPIRED` event per cancelled buff.
2. A `CANCEL_BY_TYPES` Kafka command on the existing `COMMAND_TOPIC_CHARACTER_BUFF` topic that drives the new API.
3. A producer wrapper in atlas-consumables that mirrors the existing `Apply` / `Cancel` shape.
4. A refactor of `ApplyItemEffects` so cure dispatch runs **before** HP/MP recovery (eliminates the race where a poison tick lands between drink and cancel-commit).

Two services touched. atlas-data and atlas-channel are untouched.

## Key design decisions (locked)

| ID | Decision |
|---|---|
| D1 | Cancel granularity is **whole buff**, not per-stat-change. Mob debuffs are 1:1 standalone buffs; whole-buff cancel is observably identical and preserves the existing `EXPIRED` event shape. |
| D2 | Async dispatch on `COMMAND_TOPIC_CHARACTER_BUFF` (per-character partition key). No sync REST. |
| D3 | Order in `ApplyItemEffects`: **cure → HP/MP → status buffs**. |
| D4 | atlas-consumables always emits `CANCEL_BY_TYPES` when the consumable has *any* non-zero cure spec; atlas-buffs no-ops if nothing matches. No upstream state queries. |
| D5 | Cure ignores Holy Shield. `hasImmunityBuff` gates application, not removal. |
| D6 | `thaw` is **out of scope** — it is a freeze-resistance buff (food/dish only, `0202`/`0238`), not a cure flag. |

## Files

### atlas-buffs

| Path | Change |
|---|---|
| `services/atlas-buffs/atlas.com/buffs/character/registry.go` | Add `CancelByStatTypes` method. |
| `services/atlas-buffs/atlas.com/buffs/character/registry_test.go` | New tests covering empty / no-match / single / multi-match. |
| `services/atlas-buffs/atlas.com/buffs/character/processor.go` | Add `CancelByStatTypes` to `Processor` interface and implement on `ProcessorImpl`. |
| `services/atlas-buffs/atlas.com/buffs/character/processor_test.go` | New tests for dispatch + Holy Shield ignore. |
| `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go` | Add `CommandTypeCancelByTypes` and `CancelByTypesCommandBody`. |
| `services/atlas-buffs/atlas.com/buffs/kafka/consumer/character/consumer.go` | Add `handleCancelByTypes`; register in `InitHandlers`. |
| `services/atlas-buffs/docs/domain.md` | Document new processor / registry method. |
| `services/atlas-buffs/docs/kafka.md` | Document new command type and body. |

### atlas-consumables

| Path | Change |
|---|---|
| `services/atlas-consumables/atlas.com/consumables/kafka/message/character/buff/kafka.go` | Add `CommandTypeCancelByTypes` and `CancelByTypesCommandBody`. |
| `services/atlas-consumables/atlas.com/consumables/character/buff/producer.go` | Add `cancelByTypesCommandProvider`. |
| `services/atlas-consumables/atlas.com/consumables/character/buff/processor.go` | Add `CancelByTypes(f, characterId, types)` method. |
| `services/atlas-consumables/atlas.com/consumables/consumable/processor.go` | Add `collectCureTypes(ci)` helper; reorder `ApplyItemEffects` so cure runs first. |
| `services/atlas-consumables/atlas.com/consumables/consumable/processor_test.go` | Tests for `collectCureTypes`. |
| `services/atlas-consumables/docs/domain.md` | Document the cure contract on `ApplyItemEffects` and the new buff method. |

## Conventions in this codebase

- **Module names:** `atlas-buffs` and `atlas-consumables` (short — see go.mod). Imports inside each service use the short module path (`atlas-buffs/...`, `atlas-consumables/...`).
- **Immutable models:** private fields + getters. Don't add setters.
- **Processor pattern:** `NewProcessor(l, ctx)`. Methods that emit Kafka use `message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error { ... })` for atomic emission across multiple events.
- **Tenant-scoped registries:** `tenant.MustFromContext(ctx)` then `r.characters.Get/Put(ctx, t, id)`.
- **Kafka producer (single-message):** `producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopic)(provider)`.
- **Kafka consumer:** type-tagged commands. Each handler decodes into its own body type and early-returns on type mismatch.
- **Stat type strings:** `TemporaryStatType` constants in `libs/atlas-constants/character/temporary_stat.go`. The runtime values used in `Changes()[].Type()` and the cure type list are the **string values** (`"POISON"`, `"DARKNESS"`, `"WEAKEN"`, `"SEAL"`, `"CURSE"`).

## Cure-spec → stat-type mapping

| Consumable spec (in `data/consumable.SpecType*`) | TemporaryStatType string |
|---|---|
| `SpecTypePoison` (`"poison"`) | `"POISON"` |
| `SpecTypeDarkness` (`"darkness"`) | `"DARKNESS"` |
| `SpecTypeWeakness` (`"weakness"`) | `"WEAKEN"` |
| `SpecTypeSeal` (`"seal"`) | `"SEAL"` |
| `SpecTypeCurse` (`"curse"`) | `"CURSE"` |

## Anchor functions / constants

- `services/atlas-buffs/atlas.com/buffs/character/processor.go:57` — existing `Cancel` (template for `CancelByStatTypes`).
- `services/atlas-buffs/atlas.com/buffs/character/processor.go:67` — existing `CancelAll` (closer template — emits multiple events through `message.Emit`).
- `services/atlas-buffs/atlas.com/buffs/character/registry.go:117` — existing `Cancel` registry.
- `services/atlas-buffs/atlas.com/buffs/character/registry.go:174` — existing `CancelAll` registry (template — same Get/Put-with-filtered-map shape).
- `services/atlas-buffs/atlas.com/buffs/character/producer.go:56` — `expiredStatusEventProvider`.
- `services/atlas-buffs/atlas.com/buffs/kafka/consumer/character/consumer.go:68` — existing `handleCancelAll` (template for `handleCancelByTypes`).
- `services/atlas-consumables/atlas.com/consumables/character/buff/processor.go:33` — existing `Cancel` wrapper.
- `services/atlas-consumables/atlas.com/consumables/character/buff/producer.go:41` — existing `cancelCommandProvider`.
- `services/atlas-consumables/atlas.com/consumables/consumable/processor.go:72` — `ApplyItemEffects` (the gap to close).

## Tooling / build

- Per-service Go modules. Run `go build ./...` and `go test ./...` from each service's `atlas.com/<svc>` directory.
- `setupTestRegistry(t)` in `registry_test.go` spins up `miniredis` and calls `InitRegistry(client)`. New registry tests should reuse it.
- `setupProcessorTest(t)` in `processor_test.go` returns `(Processor, tenant.Model, context.Context)`.

## Out-of-scope (do not touch)

- `SpecTypeThaw` wiring (separate task).
- Mob `Dispel` skill `127`.
- Reactor cure-all actions.
- Holy Shield application semantics.
- Any atlas-channel / atlas-data code.
