# InteractionOperationPersonalStoreRemoveItem (← `CPersonalShopDlg::MoveItemToInventory`)

- **IDA:** 0x6fdcdf
- **Atlas file:** `libs/atlas-packet/interaction/serverbound/operation_personal_store_remove_item.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | byte `` | ❌ | width mismatch |
| 1 | byte | int16 `` | ❌ | atlas: short — missing trailing field |

