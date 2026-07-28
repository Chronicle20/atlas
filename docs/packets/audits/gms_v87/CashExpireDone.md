# CashExpireDone (← `CCashShop::OnCashItemResult#EXPIRE_DONE`)

- **IDA:** 0x486981
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_misc.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x73 EXPIRE_DONE; op-byte consumed by dispatcher before CCashShop::OnCashItemResExpireDone)` | ✅ |  |
| 1 | int64 | bytes `sn: 8 bytes LARGE_INTEGER (int64), matched against m_aCashItemInfo[i].liSN before removal (same shape as DESTROY_SUCCESS)` | ✅ |  |

