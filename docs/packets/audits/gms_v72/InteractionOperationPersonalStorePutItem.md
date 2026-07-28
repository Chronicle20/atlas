# InteractionOperationPersonalStorePutItem (← `CPersonalShopDlg::PutItem`)

- **IDA:** 0x665f5f
- **Atlas file:** `libs/atlas-packet/interaction/serverbound/operation_personal_store_put_item.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `inventoryType` | ✅ |  |
| 1 | int16 | int16 `slot` | ✅ |  |
| 2 | int16 | int16 `quantity` | ✅ |  |
| 3 | int16 | int16 `set` | ✅ |  |
| 4 | int32 | int32 `price` | ✅ |  |

