# CashGiftDone (← `CCashShop::OnCashItemResult#GIFT_SUCCESS`)

- **IDA:** 0x48d3ce
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_operation_result_gift.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (0x5f GIFT_SUCCESS; op-byte consumed by dispatcher before OnCashItemResGiftDone)` | ✅ |  |
| 1 | string | string `recipientName` | ✅ |  |
| 2 | int32 | int32 `itemId (name-lookup key only; item never inserted into m_aCashItemInfo)` | ✅ |  |
| 3 | int16 | int16 `quantity` | ✅ |  |
| 4 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

