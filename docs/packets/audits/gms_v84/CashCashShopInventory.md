# CashCashShopInventory (← `CCashShop::OnCashItemResult#CashShopInventory`)

- **IDA:** 0x47c694
- **Atlas file:** `libs/atlas-packet/cash/clientbound/shop_inventory.go`
- **Variant:** GMS/v84
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int16 `` | ❌ | width mismatch |
| 1 | int16 | bytes `` | ✅ |  |
| 2 | bytes | int16 `` | ✅ |  |
| 3 | int16 | int16 `` | ✅ |  |
| 4 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |

