# CashShopOperationBuyNormal (← `CCashShop::OnBuyNormal`)

- **IDA:** 0x44cbb2
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_buy_normal.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | width mismatch |
| 1 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 2 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 3 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 4 | byte | string `` | ❌ | atlas: short — missing trailing field |

