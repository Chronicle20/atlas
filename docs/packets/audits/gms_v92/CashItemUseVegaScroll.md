# CashItemUseVegaScroll (← `CWvsContext::SendConsumeCashItemUseRequest`)

- **IDA:** 0x9bfe10
- **Atlas file:** `libs/atlas-packet/cash/serverbound/item_use_vega_scroll.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | unresolved `resolved address/decompile but zero recognized Decode/Encode calls found; see notes` | 🚫 | IDA read-order unresolved: resolved address/decompile but zero recognized Decode/Encode calls found; see notes |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

