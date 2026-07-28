# CashChangeMaplePointFailed (← `CCashShop::OnCashItemResult#CHANGE_MAPLE_POINT_FAILED`)

- **IDA:** 0x488406
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0xb0 CHANGE_MAPLE_POINT_FAILED; op-byte consumed by dispatcher before OnCashItemResChangeMaplePointFailed)` | ✅ |  |

