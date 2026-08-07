# CashRebateDone (← `CCashShop::OnCashItemResult#REBATE_SUCCESS`)

- **IDA:** 0x47e682
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_gift.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x88 REBATE_SUCCESS; op-byte consumed by dispatcher before OnCashItemResRebateDone)` | ✅ |  |
| 1 | int64 | bytes `sn (8B _LARGE_INTEGER SN; matched against existing m_aCashItemInfo[i].liSN, find-and-remove pattern)` | ✅ |  |
| 2 | int32 | int32 `amount` | ✅ |  |

