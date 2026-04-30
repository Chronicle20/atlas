# atlas-constants

Shared domain types and constants for the Atlas microservice platform.

## Why this lives here

Anything that's intrinsic to the MapleStory domain — item id ranges, inventory
types, weapon classes, world/channel/character/map ID widths, job IDs — belongs
in this library, not in any single service. When two services need the same
notion (e.g. "what's the inventory type of this item?"), one shared answer
prevents drift.

**Before defining a new domain type, alias, or numeric constant block in a
service, search this directory.** If an equivalent already exists, use it.
If you're sure nothing fits, prefer extending atlas-constants over creating
a service-local copy.

## Package index

| Package | What it provides | Use when |
|---|---|---|
| [`asset`](./asset) | `Id`, `Quantity` (item-instance ids); `Flag` bitset + `HasFlag` / `SetFlag` / `ClearFlag` | Anything dealing with concrete item instances on a character. |
| [`channel`](./channel) | `Id` (`byte`), `StatusType` | Channel identifiers in routing or socket code. **Don't redeclare as `string` or `uint8`.** |
| [`character`](./character) | `Id` (`uint32`), temporary stat constants | Character ID parameters; do not invent `int` / `string` aliases. |
| [`field`](./field) | `Id` (`string`) | Field/instance string identifiers (distinct from `map.Id`). |
| [`inventory`](./inventory) | `Type` (Equip/Use/Setup/ETC/Cash, with `.Token()`), `TypeFromItemId` | Anywhere you derive a "compartment" from an item id. **This is the canonical compartment enum — services should not reinvent it.** |
| [`inventory/slot`](./inventory/slot) | `Position`, `Type`, `Slot`, `GetSlotByType`, `GetSlotByPosition` | Equip-slot logic (cap, top, weapon, ring1…). |
| [`invite`](./invite) | `Id`, `Type`, `CommandType`, `StatusType` | Party / guild / family invite flows. |
| [`item`](./item) | `Id`, `Classification`, `WeaponType`, `GetClassification`, `GetWeaponType`, `Is*` predicates, named `Classification*` constants for hat/face/eye/earring/top/overall/bottom/shoes/gloves/shield/cape/ring/pendant/belt/medal/tamed-mob/saddle, all use+setup+etc+cash singletons | Anything that maps item id → category. **Use `item.GetClassification(id)` instead of `id / 10000`. Use the named `Classification*` constants instead of bare numeric literals.** |
| [`job`](./job) | `Id`, `Type` | Job / class IDs and type-codes (Beginner / Warrior / Magician / …). |
| [`map`](./map) | `Id` (`uint32`), field-limit constants | Map IDs in routing, drop tables, spawn rules. |
| [`monster`](./monster) | `Id`, monster status / skill constants | Monster IDs and per-monster status flags. |
| [`point`](./point) | `X`, `Y` (`int16`) | Map coordinates — keep them typed, don't pass raw ints. |
| [`skill`](./skill) | `Id`, summon-movement constants | Player and mob skill IDs. |
| [`stat`](./stat) | `Type` | Character stat keys. |
| [`world`](./world) | `Id` (`byte`) | World identifiers. **`world.Id` is `byte`, not `string` or `uint32`.** |

## Common drift symptoms (fix the cause, not the symptom)

If you spot any of these in a service, the shared type already exists here:

- `type Compartment …` or any 1..5 enum mapping equipment/use/setup/etc/cash → use `inventory.Type` (`TypeValueEquip` … `TypeValueCash`).
- `func compartmentOf(itemId)` or `itemId / 1_000_000` → use `inventory.TypeFromItemId`.
- `func classification(itemId)` or `itemId / 10_000` → use `item.GetClassification`.
- `type WorldId string` / `int` — `world.Id` is `byte`.
- `type ChannelId int` / `uint32` — `channel.Id` is `byte`.
- Bare numeric literals for known classifications (e.g. `200`, `204`, `500`) → use the `item.Classification*` named constants.
- Re-declaring weapon, monster status, or job type enums → already in `item.WeaponType`, `monster/status.go`, `job.Type`.

## Adding to atlas-constants

When extending this library:

1. Prefer adding to an existing package over creating a new one.
2. Keep additions purely additive — renaming or removing a constant is a breaking change for every consumer.
3. After adding, update the relevant row in the table above so the next contributor can find it.
