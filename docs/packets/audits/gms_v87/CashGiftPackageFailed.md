# CashGiftPackageFailed (← `CCashShop::OnCashItemResult#GIFT_PACKAGE_FAILED`)

- **IDA:** 0x4854e9
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x91 GIFT_PACKAGE_FAILED; op-byte consumed by dispatcher before OnCashItemResGiftPackageFailed)` | ✅ |  |
| 1 | byte | byte `errorCode (NoticeFailReason reason byte)` | ✅ |  |

