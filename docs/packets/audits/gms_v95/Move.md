# Move (← `CWvsContext::SendChangeSlotPositionRequest`)

- **IDA:** 0x9d9c10
- **Atlas file:** `../../libs/atlas-packet/inventory/serverbound/move.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `update_time (line 36)` | ✅ |  |
| 1 | byte | byte `nType inventoryType (line 37)` | ✅ |  |
| 2 | int16 | int16 `nOldPos source (line 38)` | ✅ |  |
| 3 | int16 | int16 `nNewPos destination (line 39)` | ✅ |  |
| 4 | int16 | int16 `nCount (line 40)` | ✅ |  |

