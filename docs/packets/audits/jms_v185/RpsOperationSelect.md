# RpsOperationSelect (← `CRPSGameDlg::SendSelection`)

- **IDA:** 0x7ae98b
- **Atlas file:** `libs/atlas-packet/rps/serverbound/operation_select.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `throw (0=Rock/1=Paper/2=Scissors, RAW passthrough; sub-op mode byte 1 written earlier by the same sender, excluded here per the storage OperationMeso/StoreAsset/RetrieveAsset convention)` | ✅ |  |

