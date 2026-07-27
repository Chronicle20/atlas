# RpsOperationSelect (← `CRPSGameDlg::SendSelection`)

- **IDA:** 0x69cb2d
- **Atlas file:** `libs/atlas-packet/rps/serverbound/operation_select.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `throw (0=Rock/1=Paper/2=Scissors, RAW passthrough; CRPSGameDlg::SendSelection @0x69cafa, COutPacket(134)+Encode1(1)+Encode1(throw) @0x69cb2d; sub-op mode byte 1 written earlier by the same sender, excluded here per the storage OperationMeso/StoreAsset/RetrieveAsset convention; live IDA port 13339 2026-07-16)` | ✅ |  |

