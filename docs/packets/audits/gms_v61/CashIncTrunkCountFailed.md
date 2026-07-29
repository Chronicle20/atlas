# CashIncTrunkCountFailed (← `CCashShop::OnCashItemResult#INC_TRUNK_COUNT_FAILED`)

- **IDA:** 0x462c65
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x47 INC_TRUNK_COUNT_FAILED; op-byte consumed by dispatcher before OnCashItemResIncTrunkCountFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (reason byte)` | ✅ |  |

