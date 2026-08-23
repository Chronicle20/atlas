# ParcelRecvNoFreeSlots (← `CParcelDlg::OnPacket#RecvNoFreeSlots`)

- **IDA:** 0x70e18d
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (21/0x15 RECV_NO_FREE_SLOTS; dispatch byte, notice-only, no body)` | ✅ |  |

