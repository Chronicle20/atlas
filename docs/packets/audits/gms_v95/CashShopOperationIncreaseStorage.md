# CashShopOperationIncreaseStorage (← `CCashShop::OnIncTrunkCount`)

- **IDA:** 0x48dc70
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_increase_storage.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `isMaplePoint bool (dwOption==2; isPoints)` | ✅ |  |
| 1 | int32 | int32 `dwOption (currency)` | ✅ |  |
| 2 | byte | byte `item bool (0 in this path -> no serialNumber)` | ✅ |  |
| 3 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

