# RpsOperationSelect (← `CRPSGameDlg::SendSelection`)

- **IDA:** 0x6c233d
- **Atlas file:** `libs/atlas-packet/rps/serverbound/operation_select.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `throw (0=Rock/1=Paper/2=Scissors, RAW passthrough; CRPSGameDlg::SendSelection @0x6c230a, COutPacket(133)+Encode1(1)+Encode1(throw) @0x6c233d; sub-op mode byte 1 written earlier by the same sender, excluded here per the storage OperationMeso/StoreAsset/RetrieveAsset convention; live IDA port 13340 2026-07-16)` | ✅ |  |

