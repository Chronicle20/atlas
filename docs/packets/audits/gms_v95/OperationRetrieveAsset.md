# OperationRetrieveAsset (← `CTrunkDlg::SendGetItemRequest`)

- **IDA:** 0x769e00
- **Atlas file:** `../../libs/atlas-packet/storage/serverbound/operation_retrieve_asset.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `inventoryType (nItemID/1000000; mode byte 4 written by dispatcher)` | ✅ |  |
| 1 | byte | byte `slot (nIdx)` | ✅ |  |

