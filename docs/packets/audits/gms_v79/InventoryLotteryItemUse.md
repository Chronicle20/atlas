# InventoryLotteryItemUse (← `CWvsContext::SendLotteryItemUseRequest`)

- **IDA:** 0x95dd02
- **Atlas file:** `libs/atlas-packet/inventory/serverbound/lottery_item_use.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `nPos (slot)` | ✅ |  |
| 1 | int32 | int32 `nItemID` | ✅ |  |

