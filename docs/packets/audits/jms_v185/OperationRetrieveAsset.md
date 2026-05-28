# OperationRetrieveAsset (← `CTrunkDlg::SendGetItemRequest`)

- **IDA:** 0x84dea0
- **Atlas file:** `../../libs/atlas-packet/storage/serverbound/operation_retrieve_asset.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nType (inventory type). op-byte 3 consumed by dispatcher` | ✅ |  |
| 1 | byte | byte `slot (*(v5+8))` | ✅ |  |

