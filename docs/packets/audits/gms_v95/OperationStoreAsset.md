# OperationStoreAsset (← `CTrunkDlg::SendPutItemRequest`)

- **IDA:** 0x768570
- **Atlas file:** `../../libs/atlas-packet/storage/serverbound/operation_store_asset.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `slot (nPOS; mode byte 5 written by dispatcher)` | ✅ |  |
| 1 | int32 | int32 `itemId (nItemID)` | ✅ |  |
| 2 | int16 | int16 `quantity (nCount)` | ✅ |  |

