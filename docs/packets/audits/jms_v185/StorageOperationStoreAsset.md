# StorageOperationStoreAsset (← `CTrunkDlg::SendPutItemRequest`)

- **IDA:** 0x84e07d
- **Atlas file:** `libs/atlas-packet/storage/serverbound/operation_store_asset.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | byte `leading sub-action byte (task-081 off-by-one remediation 2026-06-10)` | ❌ | width mismatch |
| 1 | int32 | int16 `slot (v40[0]). op-byte 4 consumed by dispatcher` | ❌ | width mismatch |
| 2 | int16 | int32 `itemId (n)` | ❌ | width mismatch |
| 3 | byte | int16 `quantity (v41[0])` | ❌ | atlas: short — missing trailing field |

