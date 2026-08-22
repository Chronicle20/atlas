# ParcelRecvUniqueConflict (← `CParcelDlg::OnPacket#RecvUniqueConflict`)

- **IDA:** 0x70e17a
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (22/0x16 RECV_UNIQUE_CONFLICT; dispatch byte, notice-only, no body)` | ✅ |  |

