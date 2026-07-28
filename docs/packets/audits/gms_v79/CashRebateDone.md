# CashRebateDone (← `CCashShop::OnCashItemResult#REBATE_SUCCESS`)

- **IDA:** 0x4743df
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_gift.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x7d REBATE_SUCCESS; op-byte consumed by dispatcher before OnCashItemResRebateDone)` | ✅ |  |
| 1 | int64 | bytes `sn (8B _LARGE_INTEGER SN; matched against existing m_aCashItemInfo[i].liSN, find-and-remove pattern)` | ✅ |  |
| 2 | int32 | int32 `amount` | ✅ |  |

