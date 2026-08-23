# ParcelRecvUniqueConflict (← `CParcelDlg::OnPacket#RecvUniqueConflict`)

- **IDA:** 0x6808f3
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (22 RECV_UNIQUE_CONFLICT; dispatch byte, notice-only, no body)` | ✅ |  |

