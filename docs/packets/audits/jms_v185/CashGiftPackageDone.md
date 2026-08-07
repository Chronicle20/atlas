# CashGiftPackageDone (← `CCashShop::OnCashItemResult#GIFT_PACKAGE_SUCCESS`)

- **IDA:** 0x48c79a
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_gift.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x8e GIFT_PACKAGE_SUCCESS; op-byte consumed by dispatcher before OnCashItemResGiftPackageDone)` | ✅ |  |
| 1 | string | string `recipientName` | ✅ |  |
| 2 | int32 | int32 `packageId (CItemInfo::GetSpecialName key)` | ✅ |  |
| 3 | int16 | int16 `unused1 (read; discarded/used only for notice-format branch client-side)` | ✅ |  |
| 4 | int16 | int16 `unused2 (read; discarded/used only for notice-format branch client-side)` | ✅ |  |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

