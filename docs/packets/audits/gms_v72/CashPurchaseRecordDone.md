# CashPurchaseRecordDone (← `CCashShop::OnCashItemResult#PURCHASE_RECORD`)

- **IDA:** 0x473ac0
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_misc.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x84 PURCHASE_RECORD; op-byte consumed by dispatcher before OnCashItemResPurchaseRecord)` | ✅ |  |
| 1 | int32 | int32 `goodsSN (used as a ZMap<long,...> key when nonzero)` | ✅ |  |
| 2 | byte | byte `purchased (byte, compared !=0 -> bool, recorded in m_mPurchaseRecord)` | ✅ |  |

