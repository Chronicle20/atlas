# MonsterMovementAck (← `CMob::OnCtrlAck`)

- **IDA:** 0x63ad65
- **Atlas file:** `libs/atlas-packet/monster/clientbound/movement_ack.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int16 `` | ❌ | width mismatch |
| 1 | int16 | byte `` | ❌ | width mismatch |
| 2 | byte | int16 `` | ❌ | width mismatch |
| 3 | int16 | byte `` | ❌ | width mismatch |
| 4 | byte | byte `` | ✅ |  |
| 5 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

