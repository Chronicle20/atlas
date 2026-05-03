# PRD — Status Cure Consumables

## Status

Investigation complete; design pending.

## Problem

Status-cure consumable items are silently no-ops. The All Cure Potion (item `2050004`) and the single-debuff cure pots (`2050000`–`2050003`) are *defined* in atlas-data with the correct cure flags (`poison`, `darkness`, `weakness`, `seal`, `curse`), but `ApplyItemEffects` in atlas-consumables never reads those flags. The character's debuffs persist through a "successful" drink — the item decrements, the cure flags surface in the parsed consumable model, then nothing happens.

## User-visible behavior

- Player is hit with mob skill `WEAKEN` (e.g. monster skill `124`/`126`) which atlas-buffs registers as a standalone buff with `sourceId = monsterSkillId` and a stat change of type `WEAKEN`.
- Player drinks an All Cure Potion (`2050004`). The item is consumed; the WEAKEN debuff remains on the player and continues to suppress stats until natural expiration.
- Same outcome for any of: `POISON`, `DARKNESS`, `SEAL`, `CURSE` debuffs and the single-effect cure pots, and for combined debuffs vs All Cure.

## Investigation findings (verified)

### Cure flags reach the consumable model

`services/atlas-data/atlas.com/data/consumable/reader.go:135-139` writes the five disease cure specs onto the consumable model:

```go
m.Spec[SpecTypePoison]   = s.GetIntegerWithDefault(string(SpecTypePoison), 0)
m.Spec[SpecTypeDarkness] = s.GetIntegerWithDefault(string(SpecTypeDarkness), 0)
m.Spec[SpecTypeWeakness] = s.GetIntegerWithDefault(string(SpecTypeWeakness), 0)
m.Spec[SpecTypeSeal]     = s.GetIntegerWithDefault(string(SpecTypeSeal), 0)
m.Spec[SpecTypeCurse]    = s.GetIntegerWithDefault(string(SpecTypeCurse), 0)
```

(The reader also parses `SpecTypeThaw` at line 134, but per a v83 WZ check `thaw` is a *buff* spec — freeze-resistance applied for the consumable's `time` — not a cure flag. It appears only on food/dish items in `Item.wz/Consume/0202.img.xml` and `0238.img.xml`, never on the `0205` cure pots. Wiring `thaw` up as a `THAW`-type buff is a separate task.)

The `SpecType*` constants are defined in `services/atlas-consumables/atlas.com/consumables/consumable/model.go:20-25` and are reachable through `ci.GetSpec(...)`.

### Per v83 WZ data (`Item.wz/Consume/0205.img.xml`)

| Item | Spec |
|---|---|
| `2050000` (Antidote) | `poison=1` |
| `2050001` (Eyedrop) | `darkness=1` |
| `2050002` (Tonic) | `weakness=1` |
| `2050003` (Holy Water) | `seal=1`, `curse=1` |
| `2050004` (All Cure Potion) | `poison=1`, `darkness=1`, `weakness=1`, `seal=1`, `curse=1` |
| `2050005` | empty `spec` — not a usable cure pot in GMS v83; out of scope |

### Consumable processor never reads them

`services/atlas-consumables/atlas.com/consumables/consumable/processor.go:72-156` `ApplyItemEffects` checks every other spec — `Accuracy`, `Evasion`, `HP`, `HPR`, `Jump`, `MagicAttack`, `MagicDefense`, `MP`, `MPR`, `WeaponAttack`, `WeaponDefense`, `Speed`, `Morph`, `Time` — but never references `SpecTypePoison`, `SpecTypeDarkness`, `SpecTypeWeakness`, `SpecTypeSeal`, or `SpecTypeCurse`. The cure flags are silently dropped. (`SpecTypeThaw` is also unreferenced, but as a non-cure buff spec it's a separate gap and out of scope here.)

### atlas-buffs cannot satisfy the cure today

`services/atlas-buffs/atlas.com/buffs/character/processor.go:57-67` exposes only:

- `Cancel(worldId, characterId, sourceId int32)` — cancel one buff by source id (e.g. monster skill id).
- `CancelAll(worldId, characterId)` — cancel every buff on the character.

There is no API that says "cancel any buff whose stat changes intersect this set of types." The consumable processor would need such an API to act on `(POISON, DARKNESS, WEAKEN, SEAL, CURSE)`.

`services/atlas-buffs/atlas.com/buffs/character/immunity.go:7-11` already enumerates the disease stat types:

```go
var diseaseStatTypes = map[string]bool{
    "STUN": true, "POISON": true, "SEAL": true, "DARKNESS": true,
    "WEAKEN": true, "CURSE": true, "SEDUCE": true, "CONFUSE": true,
    "UNDEAD": true, "SLOW": true, "STOP_PORTION": true,
}
```

### Mob debuff source-id pattern

When a mob applies a debuff (e.g. via skill `124`/`126`), atlas-monsters publishes a `BuffGive` with `sourceId = int32(monsterSkillId)` and the debuff is stored as a standalone buff in atlas-buffs keyed by that source id (`services/atlas-monsters/atlas.com/monsters/monster/processor.go:67-85`, `libs/atlas-constants/character/temporary_stat.go:36`). Each mob-applied debuff is one buff with one stat-change, so cancelling at the **buff** granularity removes the debuff cleanly without per-stat-change surgery.

## Goals

- Drinking a cure potion removes the matching debuffs from the character, in line with v83 client expectations.
- Single-debuff cure pots (`2050000`–`2050002`) cure only their specified debuff; Holy Water (`2050003`) cures Seal and Curse; All Cure (`2050004`) cures all five.
- The mechanism in atlas-buffs is general enough to support future "cure these stat types" callers (NPC clinic blessings, reactor cure-all actions, future Dispel/Heal-Self skills).
- Drinking a cure pot with no matching debuffs active continues to consume the item silently (v83 parity).
- HP/MP recovery on cure pots (e.g. All Cure also restores HP) still applies. Cure runs *before* HP/MP recovery so a poison tick cannot fire one more time after the cure.

## Non-goals

- **`thaw` buff wiring (freeze-resistance from food/dishes).** The `thaw` consumable spec is a stat-up buff applied for the consumable's `time`, not a cure flag. It appears only on food items (`Item.wz/Consume/0202.img.xml`, `0238.img.xml`), never on the cure pots in `0205`. Wiring `thaw` up as a `THAW`-type buff in `ApplyItemEffects` is a separate task.
- **Mob `Dispel` skill (skill `127`).** Already cancels player buffs via a separate path; out of scope.
- **Reactor cure-all actions.** Tracked in `docs/TODO.md`; deferred.
- **Holy Shield / immunity prevention semantics.** Already handled by `hasImmunityBuff` in atlas-buffs.
- **UI / packet changes.** atlas-channel just consumes the existing `EXPIRED` events emitted by atlas-buffs as it does today.

## Scope (preliminary — refine in design)

Two services touched, **small total scope**.

1. **atlas-buffs**
   - New processor method `CancelByStatTypes(worldId world.Id, characterId uint32, types []string) error` in `services/atlas-buffs/atlas.com/buffs/character/processor.go`. Scans the character's buffs and, for any whose `Changes()` include a stat type in `types`, runs the existing per-source `Cancel` (so the existing `EXPIRED` event is emitted unchanged).
   - Registry helper to find buffs whose changes intersect a stat-type set (`services/atlas-buffs/atlas.com/buffs/character/registry.go`).
   - New `CANCEL_BY_TYPES` Kafka command on the existing character-buff command topic, with consumer handler that calls the new processor method. Body shape:
     ```json
     {
       "worldId": 0,
       "characterId": 12345,
       "types": ["POISON", "DARKNESS", "WEAKEN", "SEAL", "CURSE"]
     }
     ```
     Single command type with a `types []string` body — flexible, future-proof for any "cure these" caller (Option A from investigation; Option B was per-type or hardcoded "all cure").
   - Unit tests: `CancelByStatTypes` with empty types, with no matching buffs, with single-match, with multi-match across multiple buffs of the same source.

2. **atlas-consumables**
   - In `ApplyItemEffects` (`services/atlas-consumables/atlas.com/consumables/consumable/processor.go:72`), after the existing spec branches, gather the non-zero cure flags and dispatch a single `CancelByTypes` via a new producer wrapper under `services/atlas-consumables/atlas.com/consumables/character/buff/`.
     - Apply order: **cure → HP/MP → status buffs.** Cure runs first so a poison/curse tick cannot fire again between drink and removal.
   - Mapping (consumable spec name → temporary stat type):
     | Consumable spec | Temporary stat type |
     |---|---|
     | `SpecTypePoison` | `POISON` |
     | `SpecTypeDarkness` | `DARKNESS` |
     | `SpecTypeWeakness` | `WEAKEN` |
     | `SpecTypeSeal` | `SEAL` |
     | `SpecTypeCurse` | `CURSE` |
   - Tests: cancel-by-types unit tests in atlas-buffs (immunity check, multi-type, no-match noop, multi-buff cancel for buffs of the same type each cancelled); `ApplyItemEffects` unit tests covering each cure flag and All Cure (multi-flag case).

3. **atlas-data** — no changes (cure flags already parsed at `consumable/reader.go:134-139`).

4. **atlas-channel** — no changes (already consumes the `EXPIRED` events that atlas-buffs emits via `Cancel`).

5. **Docs** — update `services/atlas-buffs/docs/{domain.md,kafka.md}` for the new `CancelByStatTypes` API and `CANCEL_BY_TYPES` command, and `services/atlas-consumables/docs/domain.md` for the new cure contract.

## Risks / open questions for design

- **Cancel granularity — by buff or by individual stat change?** Recommendation (a): cancel the whole buff. Mob debuffs are stored as standalone buffs keyed by `sourceId = monsterSkillId`, so the buff and the debuff are 1:1; per-stat-change surgery would complicate the existing immutable buff model and `EXPIRED` event semantics. Confirm in design.
- **Sync-vs-async dispatch.** atlas-consumables → atlas-buffs is already Kafka-async for buff give/cancel. Use the same pattern (no synchronous wait); the player's debuff removal is delivered via the existing `EXPIRED` event flow that atlas-channel already consumes. Confirm there is no UX-visible race where the player's next action (e.g. another consumable) lands before the cancel commits.
- **Apply order vs HP/MP recovery.** Today `ApplyItemEffects` applies stat-buffs first, then HP/MP. Cure should run *before* HP/MP recovery so that, e.g., a poison tick cannot fire one more time after the cure but before the heal. Confirm.
- **Holy Shield interaction.** `hasImmunityBuff` prevents new disease-type debuffs from being applied; it does not prevent existing debuffs from being cured. Cure should ignore Holy Shield (curing is desirable even if you also have Holy Shield). Verify in design.

## Success criteria

- Player hit with `WEAKEN` (mob skill `124`/`126`), drinks All Cure (`2050004`): WEAKEN buff is cancelled within one event cycle; client receives the `EXPIRED` packet; stats restore.
- Same flow with combined debuffs (e.g. WEAKEN + POISON from two mob skills): both buffs cancelled by a single drink.
- Holy Water (`2050003`) cures Seal and Curse, not Poison; the single-effect pots (`2050000`–`2050002`) cure only their specified debuff.
- All Cure also restores HP per its `HPRecovery` spec, and the order is cure-then-heal (verifiable by integration test or log trace ordering).
- Drinking a cure pot with no matching debuffs consumes the item and emits no cancel command (no spurious Kafka traffic).
- New atlas-buffs unit tests cover `CancelByStatTypes`: empty types list, no-match, single-match, multi-match across multiple buffs, immunity check.
- New atlas-consumables unit tests cover each cure flag individually and the All Cure multi-flag case, including the cure-before-HP order.
- `services/atlas-buffs/docs/{domain.md,kafka.md}` and `services/atlas-consumables/docs/domain.md` updated.

## References

- Consumable cure flag parsing: `services/atlas-data/atlas.com/data/consumable/reader.go:135-139`.
- Cure pot WZ specs: `Item.wz/Consume/0205.img.xml` (verified against v83 GMS data).
- Consumable model `SpecType*`: `services/atlas-consumables/atlas.com/consumables/consumable/model.go:20-25`.
- `ApplyItemEffects` (the gap): `services/atlas-consumables/atlas.com/consumables/consumable/processor.go:72-156`.
- atlas-buffs existing Cancel APIs: `services/atlas-buffs/atlas.com/buffs/character/processor.go:57-77`.
- Disease stat type whitelist: `services/atlas-buffs/atlas.com/buffs/character/immunity.go:7-11`.
- Mob debuff source-id convention: `services/atlas-monsters/atlas.com/monsters/monster/processor.go:67-85`, `libs/atlas-constants/character/temporary_stat.go:36`.
- Reference investigation: lost-context screenshots `/tmp/lost_context/task-051/` (re-summarised above).
