# CashGiftPackageDone (← `CCashShop::OnCashItemResult#GIFT_PACKAGE_SUCCESS`)

- **IDA:** 0x496dc0
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_gift.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x9c GIFT_PACKAGE_SUCCESS; op-byte consumed by dispatcher before OnCashItemResGiftPackageDone)` | ✅ |  |
| 1 | string | string `recipientName` | ✅ |  |
| 2 | int32 | int32 `packageId (CItemInfo::GetSpecialName key)` | ✅ |  |
| 3 | int16 | int16 `unused1 (read; discarded/used only for notice-format branch client-side)` | ✅ |  |
| 4 | int16 | int16 `unused2 (read; discarded/used only for notice-format branch client-side)` | ✅ |  |
| 5 | int32 | int32 `nxCashSpent (GMS-only; JMS build omits this trailing field — confirmed absent live)` | ✅ |  |

