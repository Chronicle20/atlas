# InteractionOperationMerchantPutItem (← `CPersonalShopDlg::PutItem#Merchant`)

- **IDA:** 0x69c880
- **Atlas file:** `../../libs/atlas-packet/interaction/serverbound/operation_merchant_put_item.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `inventoryType (nTI; op 0x21 entrusted)` | ✅ |  |
| 1 | int16 | int16 `slot (nPos)` | ✅ |  |
| 2 | int16 | int16 `quantity` | ✅ |  |
| 3 | int16 | int16 `set` | ✅ |  |
| 4 | int32 | int32 `price` | ✅ |  |

