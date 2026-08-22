# ParcelRecvUniqueConflict (← `CParcelDlg::OnPacket#RecvUniqueConflict`)

- **IDA:** 0x734ce0
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (22/0x16 RECV_UNIQUE_CONFLICT; dispatch byte, notice-only, no body)` | ✅ |  |

