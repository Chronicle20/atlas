# MonsterMovementAck (← `CMob::OnCtrlAck`)

- **IDA:** 0x66c23b
- **Atlas file:** `../../libs/atlas-packet/monster/clientbound/movement_ack.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobId — read by CMobPool::OnMobPacket before dispatch` | ✅ |  |
| 1 | int16 | int16 `mobCtrlSN` | ✅ |  |
| 2 | byte | byte `bNextAttackPossible` | ✅ |  |
| 3 | int16 | int16 `mp` | ✅ |  |
| 4 | byte | byte `skillCommand` | ✅ |  |
| 5 | byte | byte `skillLevel` | ✅ |  |

