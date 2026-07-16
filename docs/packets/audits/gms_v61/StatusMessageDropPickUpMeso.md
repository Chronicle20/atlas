# StatusMessageDropPickUpMeso (← `CWvsContext::OnMessage#DropPickUpMeso`)

- **IDA:** 0x8438b5
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v61
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `drop type 1 (meso) @0x9192f4` | ✅ |  |
| 1 | byte | int32 `meso @0x91930b` | ❌ | width mismatch |
| 2 | int32 | int16 `internetCafeBonus @0x919314` | ❌ | width mismatch |
| 3 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |

