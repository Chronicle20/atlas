# InventoryLotteryItemUse (← `CWvsContext::SendLotteryItemUseRequest`)

- **IDA:** 0xa5c8dc
- **Atlas file:** `libs/atlas-packet/inventory/serverbound/lottery_item_use.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `nPos (slot)` | ✅ |  |
| 1 | int32 | int32 `nItemID` | ✅ |  |

