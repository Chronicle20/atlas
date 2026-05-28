# OperationPersonalStorePutItem (← `CPersonalShopDlg::PutItem`)

- **IDA:** 0x69c880
- **Atlas file:** `../../libs/atlas-packet/interaction/serverbound/operation_personal_store_put_item.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `inventoryType (nTI)` | ✅ |  |
| 1 | int16 | int16 `slot (nPos)` | ✅ |  |
| 2 | int16 | int16 `quantity (count/v23)` | ✅ |  |
| 3 | int16 | int16 `set (nSet)` | ✅ |  |
| 4 | int32 | int32 `price (nPrice)` | ✅ |  |

