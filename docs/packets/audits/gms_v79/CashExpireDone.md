# CashExpireDone (← `CCashShop::OnCashItemResult#EXPIRE_DONE`)

- **IDA:** 0x4740a6
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_misc.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x66 EXPIRE_DONE; op-byte consumed by dispatcher before OnCashItemResExpireDone)` | ✅ |  |
| 1 | int64 | bytes `sn (8B _LARGE_INTEGER SN; matched against existing m_aCashItemInfo[i].liSN)` | ✅ |  |

