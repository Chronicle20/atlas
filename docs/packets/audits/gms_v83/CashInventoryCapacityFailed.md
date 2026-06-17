# CashInventoryCapacityFailed (← `CCashShop::OnCashItemResult#InventoryCapacityFailed`)

- **IDA:** 0x47aaee
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x61 INC_SLOT_COUNT_FAILED; op-byte consumed by dispatcher before OnCashItemResIncSlotCountFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (NoticeFailReason reason byte)` | ✅ |  |

