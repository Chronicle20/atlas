# InventoryItemUse (← `sub_719DD9`)

- **IDA:** 0x719dd9
- **Atlas file:** `libs/atlas-packet/inventory/serverbound/item_use.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 1 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

