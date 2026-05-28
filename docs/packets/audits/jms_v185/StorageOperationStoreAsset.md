# StorageOperationStoreAsset (← `CTrunkDlg::SendPutItemRequest`)

- **IDA:** 0x84e07d
- **Atlas file:** `../../libs/atlas-packet/storage/serverbound/operation_store_asset.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `slot (v40[0]). op-byte 4 consumed by dispatcher` | ✅ |  |
| 1 | int32 | int32 `itemId (n)` | ✅ |  |
| 2 | int16 | int16 `quantity (v41[0])` | ✅ |  |

