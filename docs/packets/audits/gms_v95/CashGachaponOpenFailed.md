# CashGachaponOpenFailed (← `CCashShop::OnCashItemResult#GACHAPON_OPEN_FAILED`)

- **IDA:** 0x4962b0
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0xb8 GACHAPON_OPEN_FAILED; op-byte consumed by dispatcher before OnCashItemResCashGachaponOpenFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (NoticeFailReason reason byte)` | ✅ |  |

