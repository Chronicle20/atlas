# ShopScannerItemUse (← `CWvsContext::SendShopScannerItemUseRequest`)

- **IDA:** 0x91e45b
- **Atlas file:** `libs/atlas-packet/merchant/serverbound/shop_scanner_item_use.go`
- **Variant:** GMS/v72
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `serial (cash-commodity serial string; legacy pre-v83 CDraggableItem::OnDoubleClicked frame)` | ✅ |  |
| 1 | int16 | int16 `source (nPOS inventory slot)` | ✅ |  |
| 2 | int32 | int32 `itemId (nItemID)` | ✅ |  |
| 3 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 7 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

