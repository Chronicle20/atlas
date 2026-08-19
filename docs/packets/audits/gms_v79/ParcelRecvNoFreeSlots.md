# ParcelRecvNoFreeSlots (← `CParcelDlg::OnPacket#RecvNoFreeSlots`)

- **IDA:** 0x683d03
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (21/0x15 RecvNoFreeSlots; dispatch byte, notice-only, no body)` | ✅ |  |

