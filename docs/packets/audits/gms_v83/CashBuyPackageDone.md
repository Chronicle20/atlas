# CashBuyPackageDone (← `CCashShop::OnCashItemResult#BUY_PACKAGE_SUCCESS`)

- **IDA:** 0x479a1b
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_gift.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x89 BUY_PACKAGE_SUCCESS; op-byte consumed by dispatcher before OnCashItemResBuyPackageDone)` | ✅ |  |
| 1 | byte | byte `itemCount (byte, count-prefix for the item list)` | ✅ |  |
| 2 | bytes | bytes `GW_CashItemInfo entry (55B CashInventoryItem blob)` | ✅ |  |
| 3 | int16 | int16 `trailingCount (uint16; branches notice-text format client-side)` | ✅ |  |

