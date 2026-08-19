# ParcelRecvEnableActions (← `CParcelDlg::OnPacket#RecvEnableActions`)

- **IDA:** 0x683cd8
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (20/0x14 RecvEnableActions; dispatch byte, notice-only, no body)` | ✅ |  |

