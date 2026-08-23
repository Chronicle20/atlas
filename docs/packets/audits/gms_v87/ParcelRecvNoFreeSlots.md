# ParcelRecvNoFreeSlots (← `CParcelDlg::OnPacket#RecvNoFreeSlots`)

- **IDA:** 0x734cf3
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (21/0x15 RECV_NO_FREE_SLOTS; dispatch byte, notice-only, no body)` | ✅ |  |

