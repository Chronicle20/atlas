# OperationError (← `CCashShop::OnCashItemResult#OperationError`)

- **IDA:** 0x4969f0
- **Atlas file:** `../../libs/atlas-packet/cash/clientbound/shop_operation_result.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x65 BUY_FAILED; op-byte consumed by dispatcher; representative of failure family)` | ✅ |  |
| 1 | byte | byte `errorCode (NoticeFailReason reason byte)` | ✅ |  |

