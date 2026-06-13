# InventoryMove (← `CWvsContext::SendChangeSlotPositionRequest`)

- **IDA:** 0xaeda01
- **Atlas file:** `libs/atlas-packet/inventory/serverbound/move.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | byte | byte `` | ✅ |  |
| 2 | int16 | int16 `` | ✅ |  |
| 3 | int16 | int16 `` | ✅ |  |
| 4 | int16 | int16 `` | ✅ |  |

