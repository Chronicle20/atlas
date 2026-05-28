# MonsterDamageFriendly (← `CMob::Update`)

- **IDA:** N/A — send-side function `CMob::Update` not present in gms_v95.json export
- **Atlas file:** `libs/atlas-packet/character/serverbound/monster_damage_friendly.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `attackerId` | 🔍 | no IDA export — shape inferred from ServerBound CSV (CMob::Update) |
| 1 | int32 | int32 `observerId` | 🔍 | no IDA export |
| 2 | int32 | int32 `attackedId` | 🔍 | no IDA export |

## Notes

ack: no IDA export — `CMob::Update` (the CSV-listed FName for MOB_DAMAGE_MOB_FRIENDLY opcode 0xE7/231 in GMS v95) was not found in the IDA binary. The atlas implementation (3 × uint32: attackerId, observerId, attackedId) is consistent with the packet semantics (the client reports which friendly mob [attackerId] observed to be attacking another mob [attackedId] while the local character is the observer [observerId]). Verdict 🔍 pending a future IDA export of CMob::Update.
