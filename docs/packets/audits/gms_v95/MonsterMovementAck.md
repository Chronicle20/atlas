# MonsterMovementAck (← `CMob::OnCtrlAck`)

- **IDA:** 0x640c50
- **Atlas file:** `../../libs/atlas-packet/monster/clientbound/movement_ack.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobId — read by CMobPool::OnMobPacket before dispatch` | ✅ |  |
| 1 | int16 | int16 `mobCtrlSN (v5 — int16)` | ✅ |  |
| 2 | byte | byte `bNextAttackPossible` | ✅ |  |
| 3 | int16 | int16 `mp (uint16)` | ✅ |  |
| 4 | byte | byte `skillCommand (v7)` | ✅ |  |
| 5 | byte | byte `skillLevel (v8)` | ✅ |  |

